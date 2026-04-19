package batchflow

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go-fiber-core/internal/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type manager struct {
	lifecycle    ParentLifecycle
	dataProvider DataProvider
	processor    BatchProcessor
	finalizer    Finalizer
	stateStore   StateStore
	defaultTTL   time.Duration
	runControl   *RunControl
}

func NewManager(
	lifecycle ParentLifecycle,
	dataProvider DataProvider,
	processor BatchProcessor,
	finalizer Finalizer,
	stateStore StateStore,
	defaultTTL time.Duration,
	runControl *RunControl,
) Manager {
	if defaultTTL <= 0 {
		defaultTTL = 24 * time.Hour
	}
	return &manager{
		lifecycle:    lifecycle,
		dataProvider: dataProvider,
		processor:    processor,
		finalizer:    finalizer,
		stateStore:   stateStore,
		defaultTTL:   defaultTTL,
		runControl:   runControl,
	}
}

func (m *manager) Start(ctx context.Context, req StartRequest) (StartResult, error) {
	if err := validateStartInput(req.Input); err != nil {
		return StartResult{}, err
	}
	if strings.TrimSpace(req.Input.RedisKey) == "" {
		req.Input.RedisKey = fmt.Sprintf("run-%s", uuid.NewString())
	}
	if req.BatchSize <= 0 {
		req.BatchSize = 5000
	}
	if req.RedisTTL <= 0 {
		req.RedisTTL = m.defaultTTL
	}

	execCtx := m.newExecutionContext(req.Input, Summary{}, req.RedisTTL)
	if m.lifecycle != nil {
		if err := m.lifecycle.Start(ctx, execCtx); err != nil {
			return StartResult{}, err
		}
	}

	load, err := m.dataProvider.LoadBatches(ctx, execCtx, req.BatchSize)
	if err != nil {
		return StartResult{}, err
	}

	batchesListKey, err := m.stateStore.Initialize(ctx, req.Input, load.Batches, load.Summary, load.Metadata, req.RedisTTL)
	if err != nil {
		return StartResult{}, err
	}
	if m.runControl != nil {
		if err := m.runControl.RegisterRun(ctx, req.Input, req.RedisTTL); err != nil {
			return StartResult{}, err
		}
	}

	totalBatches := len(load.Batches)
	if totalBatches == 0 {
		totalBatches = 1
	}

	if execCtx.Runtime != nil {
		_ = execCtx.Runtime.Set(ctx, "summary", load.Summary)
		_ = execCtx.Runtime.Set(ctx, "metadata", load.Metadata)
	}

	return StartResult{
		RedisKey:       req.Input.RedisKey,
		BatchesListKey: batchesListKey,
		TotalBatches:   totalBatches,
		Summary:        load.Summary,
		Metadata:       load.Metadata,
	}, nil
}

func (m *manager) DispatchShards(ctx context.Context, req DispatchRequest) (DispatchResult, error) {
	if err := validateInput(req.Input); err != nil {
		return DispatchResult{}, err
	}
	if req.TotalBatches <= 0 {
		return DispatchResult{}, fmt.Errorf("total_batches inválido")
	}
	if req.ParallelShards <= 0 {
		req.ParallelShards = 1
	}
	if req.ParallelShards > req.TotalBatches {
		req.ParallelShards = req.TotalBatches
	}
	if err := m.stateStore.RegisterShards(ctx, req.Input, req.ParallelShards, m.defaultTTL); err != nil {
		return DispatchResult{}, err
	}

	indexes := make([]int, 0, req.ParallelShards)
	for shardIndex := 0; shardIndex < req.ParallelShards; shardIndex++ {
		indexes = append(indexes, shardIndex)
	}

	return DispatchResult{
		TotalShards:         req.ParallelShards,
		InitialBatchIndexes: indexes,
	}, nil
}

func (m *manager) ProcessBatch(ctx context.Context, req ProcessRequest) (ProcessResult, error) {
	if err := validateInput(req.Input); err != nil {
		return ProcessResult{}, err
	}
	if cancelled, status, err := m.runCancellationStatus(ctx, req.Input); err != nil {
		return ProcessResult{}, err
	} else if cancelled {
		if lockErr := m.handleCancelledRun(ctx, req.Input, status); lockErr != nil {
			return ProcessResult{}, lockErr
		}
		return cancelledProcessResult(req, status), nil
	}
	if req.TotalShards <= 0 {
		req.TotalShards = 1
	}
	if req.ShardIndex < 0 || req.ShardIndex >= req.TotalShards {
		return ProcessResult{}, fmt.Errorf("shard_index fuera de rango: %d (total_shards=%d)", req.ShardIndex, req.TotalShards)
	}
	if req.BatchIndex < 0 || req.BatchIndex >= req.TotalBatches {
		return ProcessResult{}, fmt.Errorf("batch_index fuera de rango: %d (total=%d)", req.BatchIndex, req.TotalBatches)
	}
	if req.ConcurrentBatches <= 0 {
		req.ConcurrentBatches = 1
	}

	batchIndexes := make([]int, 0, req.ConcurrentBatches)
	for offset := 0; offset < req.ConcurrentBatches; offset++ {
		batchIndex := req.BatchIndex + (offset * req.TotalShards)
		if batchIndex >= req.TotalBatches {
			break
		}
		batchIndexes = append(batchIndexes, batchIndex)
	}
	if len(batchIndexes) == 0 {
		return ProcessResult{}, fmt.Errorf("no hay batches para shard=%d desde batch_index=%d", req.ShardIndex, req.BatchIndex)
	}

	summary, err := m.stateStore.LoadSummary(ctx, req.Input)
	if err != nil {
		return ProcessResult{}, err
	}
	execCtx := m.newExecutionContext(req.Input, summary, m.defaultTTL)

	type batchResult struct {
		index int
		data  ProcessBatchResult
		err   error
	}

	if req.DispatchPacing.Enabled {
		batch, loadErr := m.stateStore.LoadBatch(ctx, req.BatchesListKey, req.BatchIndex)
		if loadErr != nil {
			return ProcessResult{}, loadErr
		}
		pacingRes, processErr := ProcessBatchWithDispatchPacing(ctx, m.processor, execCtx, batch, req.BatchIndex, req.DispatchPacing, req.TotalShards)
		if processErr != nil {
			return ProcessResult{}, processErr
		}
		totalProcessed := pacingRes.ProcessResult.ProcessedCount
		metadata := map[string]any{
			fmt.Sprintf("batch_%d", req.BatchIndex): pacingRes.ProcessResult.Metadata,
		}
		nextIndex := pacingRes.NextBatchIndex
		isShardComplete := pacingRes.BatchComplete && nextIndex >= req.TotalBatches
		isLast := isShardComplete && req.TotalShards == 1

		var completedShards int64
		shouldDispatchNextStep := false
		if isShardComplete {
			completion, err := m.stateStore.CompleteShard(ctx, req.Input, req.ShardIndex, req.TotalShards, m.defaultTTL)
			if err != nil {
				return ProcessResult{}, err
			}
			completedShards = completion.CompletedShards
			shouldDispatchNextStep = completion.ShouldFinalize
		}

		return ProcessResult{
			NextBatchIndex:         nextIndex,
			IsLastBatch:            isLast,
			IsShardComplete:        isShardComplete,
			ProcessedCount:         totalProcessed,
			BatchesProcessed:       1,
			ShardIndex:             req.ShardIndex,
			TotalShards:            req.TotalShards,
			CompletedShards:        completedShards,
			ShouldDispatchNextStep: shouldDispatchNextStep,
			Metadata:               metadata,
		}, nil
	}

	results := make(chan batchResult, len(batchIndexes))
	{
		var wg sync.WaitGroup
		for _, batchIndex := range batchIndexes {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				batch, loadErr := m.stateStore.LoadBatch(ctx, req.BatchesListKey, idx)
				if loadErr != nil {
					results <- batchResult{index: idx, err: loadErr}
					return
				}
				res, processErr := m.processor.ProcessBatch(ctx, execCtx, batch)
				results <- batchResult{index: idx, data: res, err: processErr}
			}(batchIndex)
		}
		wg.Wait()
	}
	close(results)

	totalProcessed := 0
	metadata := make(map[string]any)
	for result := range results {
		if result.err != nil {
			return ProcessResult{}, result.err
		}
		totalProcessed += result.data.ProcessedCount
		metadata[fmt.Sprintf("batch_%d", result.index)] = result.data.Metadata
	}

	nextIndex := req.BatchIndex + (req.ConcurrentBatches * req.TotalShards)
	isShardComplete := nextIndex >= req.TotalBatches
	isLast := isShardComplete && req.TotalShards == 1

	var completedShards int64
	shouldDispatchNextStep := false
	if isShardComplete {
		completion, err := m.stateStore.CompleteShard(ctx, req.Input, req.ShardIndex, req.TotalShards, m.defaultTTL)
		if err != nil {
			return ProcessResult{}, err
		}
		completedShards = completion.CompletedShards
		shouldDispatchNextStep = completion.ShouldFinalize
	}

	return ProcessResult{
		NextBatchIndex:         nextIndex,
		IsLastBatch:            isLast,
		IsShardComplete:        isShardComplete,
		ProcessedCount:         totalProcessed,
		BatchesProcessed:       len(batchIndexes),
		ShardIndex:             req.ShardIndex,
		TotalShards:            req.TotalShards,
		CompletedShards:        completedShards,
		ShouldDispatchNextStep: shouldDispatchNextStep,
		Metadata:               metadata,
	}, nil
}

func (m *manager) Finalize(ctx context.Context, req FinalizeRequest) (FinalizeResult, error) {
	if err := validateInput(req.Input); err != nil {
		return FinalizeResult{}, err
	}
	if cancelled, status, err := m.runCancellationStatus(ctx, req.Input); err != nil {
		return FinalizeResult{}, err
	} else if cancelled {
		logger.LogExecutionGuard(
			"run_finalize_skipped_cancelled",
			zap.String("run_key", req.Input.RedisKey),
			zap.Int64("parent_id", req.Input.ParentID),
			zap.String("reason", status.Reason),
		)
		return FinalizeResult{
			Summary: Summary{},
			Metadata: map[string]any{
				"cancelled":     true,
				"cancel_reason": status.Reason,
			},
		}, nil
	}

	summary, err := m.stateStore.LoadSummary(ctx, req.Input)
	if err != nil {
		return FinalizeResult{}, err
	}

	execCtx := m.newExecutionContext(req.Input, summary, m.defaultTTL)
	result := FinalizeResult{
		Summary: summary,
	}
	if m.finalizer != nil {
		result, err = m.finalizer.Finalize(ctx, execCtx, req)
		if err != nil {
			return FinalizeResult{}, err
		}
	}
	if m.lifecycle != nil {
		if err := m.lifecycle.End(ctx, execCtx, result); err != nil {
			return FinalizeResult{}, err
		}
	}
	if err := m.stateStore.Cleanup(ctx, req.Input, req.BatchesListKey); err != nil {
		return FinalizeResult{}, err
	}
	if m.runControl != nil {
		_ = m.runControl.MarkCompleted(ctx, req.Input)
	}
	return result, nil
}

func (m *manager) Fail(ctx context.Context, input Input, cause error) error {
	if m.runControl != nil {
		_ = m.runControl.MarkFailed(ctx, input, cause)
	}
	if m.lifecycle == nil {
		return nil
	}
	return m.lifecycle.Fail(ctx, m.newExecutionContext(input, Summary{}, m.defaultTTL), cause)
}

func validateInput(input Input) error {
	if err := validateStartInput(input); err != nil {
		return err
	}
	if strings.TrimSpace(input.RedisKey) == "" {
		return fmt.Errorf("key_redis inválida")
	}
	return nil
}

func validateStartInput(input Input) error {
	if input.ParentID <= 0 {
		return fmt.Errorf("id inválido")
	}
	return nil
}

func (m *manager) newExecutionContext(input Input, summary Summary, ttl time.Duration) ExecutionContext {
	execCtx := ExecutionContext{
		Input:   input,
		Summary: summary,
	}
	if provider, ok := m.stateStore.(RuntimeValuesProvider); ok {
		execCtx.Runtime = provider.RuntimeValues(input, ttl)
	}
	return execCtx
}

func (m *manager) runCancellationStatus(ctx context.Context, input Input) (bool, RunStatus, error) {
	if m.runControl == nil || strings.TrimSpace(input.RedisKey) == "" {
		return false, RunStatus{}, nil
	}
	return m.runControl.IsCancelled(ctx, input.RedisKey)
}

func (m *manager) handleCancelledRun(ctx context.Context, input Input, status RunStatus) error {
	if m.runControl == nil || strings.TrimSpace(input.RedisKey) == "" {
		return nil
	}
	locked, err := m.runControl.AcquireStopLock(ctx, input.RedisKey, m.defaultTTL)
	if err != nil {
		return err
	}
	if !locked {
		return nil
	}
	logger.LogExecutionGuard(
		"run_cancel_detected",
		zap.String("run_key", input.RedisKey),
		zap.Int64("parent_id", input.ParentID),
		zap.String("reason", status.Reason),
		zap.String("source", status.Source),
	)
	if m.lifecycle != nil {
		return m.lifecycle.Fail(ctx, m.newExecutionContext(input, Summary{}, m.defaultTTL), ErrRunCancelled)
	}
	return nil
}

func cancelledProcessResult(req ProcessRequest, status RunStatus) ProcessResult {
	return ProcessResult{
		NextBatchIndex:         req.BatchIndex,
		IsLastBatch:            true,
		IsShardComplete:        true,
		ProcessedCount:         0,
		BatchesProcessed:       0,
		ShardIndex:             req.ShardIndex,
		TotalShards:            req.TotalShards,
		CompletedShards:        0,
		ShouldDispatchNextStep: false,
		Metadata: map[string]any{
			"cancelled":     true,
			"cancel_reason": status.Reason,
			"cancel_source": status.Source,
		},
	}
}
