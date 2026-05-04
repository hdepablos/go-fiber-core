package batchflow

import (
	"context"
	"fmt"
)

const defaultCursorBatchSize = 5000

func (m *manager) startCursorRun(ctx context.Context, execCtx ExecutionContext, req StartRequest) (LoadBatchesResult, string, error) {
	provider, ok := m.dataProvider.(CursorDataProvider)
	if !ok {
		return LoadBatchesResult{}, "", fmt.Errorf("data provider no soporta source_mode=%s", req.SourceMode)
	}

	cursorRun, err := provider.PrepareCursorRun(ctx, execCtx, req.BatchSize)
	if err != nil {
		return LoadBatchesResult{}, "", err
	}

	batchesListKey, err := m.stateStore.Initialize(ctx, req.Input, nil, cursorRun.Summary, cursorRun.Metadata, req.RedisTTL)
	if err != nil {
		return LoadBatchesResult{}, "", err
	}

	if execCtx.Runtime != nil {
		if err := execCtx.Runtime.Set(ctx, "source_mode", SourceModeCursor); err != nil {
			return LoadBatchesResult{}, "", err
		}
		if err := execCtx.Runtime.Set(ctx, "cursor_batch_size", req.BatchSize); err != nil {
			return LoadBatchesResult{}, "", err
		}
		if err := saveCursorState(ctx, execCtx.Runtime, 0, cursorRun.InitialCursor); err != nil {
			return LoadBatchesResult{}, "", err
		}
	}

	return LoadBatchesResult{
		Summary:  cursorRun.Summary,
		Metadata: cursorRun.Metadata,
	}, batchesListKey, nil
}

func (m *manager) processCursorBatch(ctx context.Context, req ProcessRequest) (ProcessResult, error) {
	if req.TotalShards <= 0 {
		req.TotalShards = 1
	}
	if req.TotalShards > 1 {
		return ProcessResult{}, fmt.Errorf("source_mode=%s no soporta parallel_shards > 1", req.SourceMode)
	}
	if req.ConcurrentBatches <= 0 {
		req.ConcurrentBatches = 1
	}

	summary, err := m.stateStore.LoadSummary(ctx, req.Input)
	if err != nil {
		return ProcessResult{}, err
	}
	execCtx := m.newExecutionContext(req.Input, summary, m.defaultTTL)
	if execCtx.Runtime == nil {
		return ProcessResult{}, fmt.Errorf("runtime values no disponibles para source_mode=%s", req.SourceMode)
	}

	provider, ok := m.dataProvider.(CursorDataProvider)
	if !ok {
		return ProcessResult{}, fmt.Errorf("data provider no soporta source_mode=%s", req.SourceMode)
	}

	batchSize, err := loadCursorBatchSize(ctx, execCtx.Runtime)
	if err != nil {
		return ProcessResult{}, err
	}
	progressRefresher := batchProgressRefresherFromLifecycle(m.lifecycle)

	if req.DispatchPacing.Enabled {
		return m.processCursorBatchWithPacing(ctx, execCtx, provider, progressRefresher, req, batchSize)
	}

	currentCursor, err := loadCursorState(ctx, execCtx.Runtime, req.ShardIndex)
	if err != nil {
		return ProcessResult{}, err
	}

	totalProcessed := 0
	batchesProcessed := 0
	metadata := make(map[string]any)
	currentBatchIndex := req.BatchIndex

	for offset := 0; offset < req.ConcurrentBatches; offset++ {
		loadRes, err := provider.LoadCursorBatch(ctx, execCtx, CursorBatchRequest{
			BatchSize:   batchSize,
			BatchIndex:  currentBatchIndex,
			ShardIndex:  req.ShardIndex,
			TotalShards: req.TotalShards,
			Cursor:      currentCursor,
		})
		if err != nil {
			return ProcessResult{}, err
		}
		if len(loadRes.Batch.Items) == 0 {
			if loadRes.HasMore {
				return ProcessResult{}, fmt.Errorf("cursor batch vacío con has_more=true en batch_index=%d", currentBatchIndex)
			}
			return m.completeCursorShard(ctx, req, currentBatchIndex, totalProcessed, batchesProcessed, metadata)
		}

		processRes, err := m.processor.ProcessBatch(ctx, execCtx, loadRes.Batch)
		if err == nil {
			err = refreshBatchProgress(ctx, progressRefresher, execCtx, loadRes.Batch)
		}
		if err != nil {
			return ProcessResult{}, err
		}

		totalProcessed += processRes.ProcessedCount
		batchesProcessed++
		metadata[fmt.Sprintf("batch_%d", currentBatchIndex)] = buildCursorBatchMetadata(loadRes.Metadata, processRes.Metadata, currentCursor, loadRes.NextCursor)

		if err := saveCursorState(ctx, execCtx.Runtime, req.ShardIndex, loadRes.NextCursor); err != nil {
			return ProcessResult{}, err
		}
		currentCursor = cloneCursorState(loadRes.NextCursor)
		currentBatchIndex++

		if !loadRes.HasMore {
			return m.completeCursorShard(ctx, req, currentBatchIndex, totalProcessed, batchesProcessed, metadata)
		}
	}

	return ProcessResult{
		NextBatchIndex:         currentBatchIndex,
		IsLastBatch:            false,
		IsShardComplete:        false,
		ProcessedCount:         totalProcessed,
		BatchesProcessed:       batchesProcessed,
		ShardIndex:             req.ShardIndex,
		TotalShards:            req.TotalShards,
		CompletedShards:        0,
		ShouldDispatchNextStep: false,
		Metadata:               metadata,
	}, nil
}

func (m *manager) processCursorBatchWithPacing(
	ctx context.Context,
	execCtx ExecutionContext,
	provider CursorDataProvider,
	progressRefresher BatchProgressRefresher,
	req ProcessRequest,
	batchSize int,
) (ProcessResult, error) {
	batch, nextCursor, hasMore, providerMetadata, hasPending, err := loadPendingCursorBatch(ctx, execCtx.Runtime, req.ShardIndex, req.BatchIndex)
	if err != nil {
		return ProcessResult{}, err
	}
	if !hasPending {
		currentCursor, err := loadCursorState(ctx, execCtx.Runtime, req.ShardIndex)
		if err != nil {
			return ProcessResult{}, err
		}
		loadRes, err := provider.LoadCursorBatch(ctx, execCtx, CursorBatchRequest{
			BatchSize:   batchSize,
			BatchIndex:  req.BatchIndex,
			ShardIndex:  req.ShardIndex,
			TotalShards: req.TotalShards,
			Cursor:      currentCursor,
		})
		if err != nil {
			return ProcessResult{}, err
		}
		if len(loadRes.Batch.Items) == 0 {
			if loadRes.HasMore {
				return ProcessResult{}, fmt.Errorf("cursor batch vacío con has_more=true en batch_index=%d", req.BatchIndex)
			}
			return m.completeCursorShard(ctx, req, req.BatchIndex, 0, 0, map[string]any{
				fmt.Sprintf("batch_%d", req.BatchIndex): buildCursorBatchMetadata(loadRes.Metadata, nil, currentCursor, loadRes.NextCursor),
			})
		}
		if err := savePendingCursorBatch(ctx, execCtx.Runtime, req.ShardIndex, req.BatchIndex, loadRes.Batch, loadRes.NextCursor, loadRes.HasMore, loadRes.Metadata); err != nil {
			return ProcessResult{}, err
		}
		batch = loadRes.Batch
		nextCursor = loadRes.NextCursor
		hasMore = loadRes.HasMore
		providerMetadata = loadRes.Metadata
	}

	pacingRes, err := ProcessBatchWithDispatchPacing(ctx, m.processor, progressRefresher, execCtx, batch, req.BatchIndex, req.DispatchPacing, req.TotalShards)
	if err != nil {
		return ProcessResult{}, err
	}

	metadata := map[string]any{
		fmt.Sprintf("batch_%d", req.BatchIndex): buildCursorBatchMetadata(providerMetadata, pacingRes.ProcessResult.Metadata, nil, nextCursor),
	}
	if !pacingRes.BatchComplete {
		return ProcessResult{
			NextBatchIndex:         pacingRes.NextBatchIndex,
			IsLastBatch:            false,
			IsShardComplete:        false,
			ProcessedCount:         pacingRes.ProcessResult.ProcessedCount,
			BatchesProcessed:       1,
			ShardIndex:             req.ShardIndex,
			TotalShards:            req.TotalShards,
			CompletedShards:        0,
			ShouldDispatchNextStep: false,
			Metadata:               metadata,
		}, nil
	}

	if err := saveCursorState(ctx, execCtx.Runtime, req.ShardIndex, nextCursor); err != nil {
		return ProcessResult{}, err
	}
	if err := deletePendingCursorBatch(ctx, execCtx.Runtime, req.ShardIndex, req.BatchIndex); err != nil {
		return ProcessResult{}, err
	}
	if !hasMore {
		return m.completeCursorShard(ctx, req, pacingRes.NextBatchIndex, pacingRes.ProcessResult.ProcessedCount, 1, metadata)
	}

	return ProcessResult{
		NextBatchIndex:         pacingRes.NextBatchIndex,
		IsLastBatch:            false,
		IsShardComplete:        false,
		ProcessedCount:         pacingRes.ProcessResult.ProcessedCount,
		BatchesProcessed:       1,
		ShardIndex:             req.ShardIndex,
		TotalShards:            req.TotalShards,
		CompletedShards:        0,
		ShouldDispatchNextStep: false,
		Metadata:               metadata,
	}, nil
}

func (m *manager) completeCursorShard(ctx context.Context, req ProcessRequest, nextBatchIndex int, totalProcessed int, batchesProcessed int, metadata map[string]any) (ProcessResult, error) {
	completion, err := m.stateStore.CompleteShard(ctx, req.Input, req.ShardIndex, req.TotalShards, m.defaultTTL)
	if err != nil {
		return ProcessResult{}, err
	}

	return ProcessResult{
		NextBatchIndex:         nextBatchIndex,
		IsLastBatch:            req.TotalShards == 1,
		IsShardComplete:        true,
		ProcessedCount:         totalProcessed,
		BatchesProcessed:       batchesProcessed,
		ShardIndex:             req.ShardIndex,
		TotalShards:            req.TotalShards,
		CompletedShards:        completion.CompletedShards,
		ShouldDispatchNextStep: completion.ShouldFinalize,
		Metadata:               metadata,
	}, nil
}

func loadCursorBatchSize(ctx context.Context, runtime RuntimeValues) (int, error) {
	if runtime == nil {
		return defaultCursorBatchSize, nil
	}
	var batchSize int
	if err := runtime.Get(ctx, "cursor_batch_size", &batchSize); err != nil || batchSize <= 0 {
		return defaultCursorBatchSize, nil
	}
	return batchSize, nil
}

func loadCursorState(ctx context.Context, runtime RuntimeValues, shardIndex int) (map[string]any, error) {
	if runtime == nil {
		return map[string]any{}, nil
	}
	var cursor map[string]any
	if err := runtime.Get(ctx, cursorStateRuntimeKey(shardIndex), &cursor); err != nil {
		return map[string]any{}, nil
	}
	return cloneCursorState(cursor), nil
}

func saveCursorState(ctx context.Context, runtime RuntimeValues, shardIndex int, cursor map[string]any) error {
	if runtime == nil {
		return nil
	}
	if cursor == nil {
		cursor = map[string]any{}
	}
	return runtime.Set(ctx, cursorStateRuntimeKey(shardIndex), cursor)
}

func loadPendingCursorBatch(ctx context.Context, runtime RuntimeValues, shardIndex int, batchIndex int) (Batch, map[string]any, bool, map[string]any, bool, error) {
	if runtime == nil {
		return Batch{}, nil, false, nil, false, nil
	}

	var batch Batch
	if err := runtime.Get(ctx, pendingCursorBatchRuntimeKey(shardIndex, batchIndex), &batch); err != nil {
		return Batch{}, nil, false, nil, false, nil
	}

	var nextCursor map[string]any
	if err := runtime.Get(ctx, pendingCursorNextCursorRuntimeKey(shardIndex, batchIndex), &nextCursor); err != nil {
		return Batch{}, nil, false, nil, false, err
	}
	var hasMore bool
	if err := runtime.Get(ctx, pendingCursorHasMoreRuntimeKey(shardIndex, batchIndex), &hasMore); err != nil {
		return Batch{}, nil, false, nil, false, err
	}
	var metadata map[string]any
	if err := runtime.Get(ctx, pendingCursorMetadataRuntimeKey(shardIndex, batchIndex), &metadata); err != nil {
		metadata = nil
	}

	return batch, nextCursor, hasMore, metadata, true, nil
}

func savePendingCursorBatch(ctx context.Context, runtime RuntimeValues, shardIndex int, batchIndex int, batch Batch, nextCursor map[string]any, hasMore bool, metadata map[string]any) error {
	if runtime == nil {
		return nil
	}
	if err := runtime.Set(ctx, pendingCursorBatchRuntimeKey(shardIndex, batchIndex), batch); err != nil {
		return err
	}
	if err := runtime.Set(ctx, pendingCursorNextCursorRuntimeKey(shardIndex, batchIndex), cloneCursorState(nextCursor)); err != nil {
		return err
	}
	if err := runtime.Set(ctx, pendingCursorHasMoreRuntimeKey(shardIndex, batchIndex), hasMore); err != nil {
		return err
	}
	return runtime.Set(ctx, pendingCursorMetadataRuntimeKey(shardIndex, batchIndex), metadata)
}

func deletePendingCursorBatch(ctx context.Context, runtime RuntimeValues, shardIndex int, batchIndex int) error {
	if runtime == nil {
		return nil
	}
	for _, key := range []string{
		pendingCursorBatchRuntimeKey(shardIndex, batchIndex),
		pendingCursorNextCursorRuntimeKey(shardIndex, batchIndex),
		pendingCursorHasMoreRuntimeKey(shardIndex, batchIndex),
		pendingCursorMetadataRuntimeKey(shardIndex, batchIndex),
	} {
		if err := runtime.Delete(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

func buildCursorBatchMetadata(providerMetadata map[string]any, processMetadata map[string]any, cursor map[string]any, nextCursor map[string]any) map[string]any {
	metadata := map[string]any{}
	if len(providerMetadata) > 0 {
		metadata["provider"] = providerMetadata
	}
	if len(processMetadata) > 0 {
		metadata["processor"] = processMetadata
	}
	if len(cursor) > 0 {
		metadata["cursor"] = cursor
	}
	if len(nextCursor) > 0 {
		metadata["next_cursor"] = nextCursor
	}
	return metadata
}

func cloneCursorState(cursor map[string]any) map[string]any {
	if len(cursor) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(cursor))
	for k, v := range cursor {
		cloned[k] = v
	}
	return cloned
}

func cursorStateRuntimeKey(shardIndex int) string {
	return fmt.Sprintf("cursor:shard:%d:state", shardIndex)
}

func pendingCursorBatchRuntimeKey(shardIndex int, batchIndex int) string {
	return fmt.Sprintf("cursor:shard:%d:batch:%d:payload", shardIndex, batchIndex)
}

func pendingCursorNextCursorRuntimeKey(shardIndex int, batchIndex int) string {
	return fmt.Sprintf("cursor:shard:%d:batch:%d:next_cursor", shardIndex, batchIndex)
}

func pendingCursorHasMoreRuntimeKey(shardIndex int, batchIndex int) string {
	return fmt.Sprintf("cursor:shard:%d:batch:%d:has_more", shardIndex, batchIndex)
}

func pendingCursorMetadataRuntimeKey(shardIndex int, batchIndex int) string {
	return fmt.Sprintf("cursor:shard:%d:batch:%d:metadata", shardIndex, batchIndex)
}
