package batchflow

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type cursorTestProvider struct {
	batches      []Batch
	prepareCalls int
	loadCalls    int
}

func (p *cursorTestProvider) LoadBatches(context.Context, ExecutionContext, int) (LoadBatchesResult, error) {
	return LoadBatchesResult{}, fmt.Errorf("LoadBatches no esperado en source_mode=cursor")
}

func (p *cursorTestProvider) PrepareCursorRun(_ context.Context, _ ExecutionContext, _ int) (CursorRunResult, error) {
	p.prepareCalls++
	totalRecords := 0
	for _, batch := range p.batches {
		totalRecords += len(batch.Items)
	}
	return CursorRunResult{
		Summary: Summary{
			TotalRecords: int64(totalRecords),
			Metadata: map[string]any{
				"source_mode": SourceModeCursor,
			},
		},
		Metadata: map[string]any{
			"source_mode": SourceModeCursor,
		},
		InitialCursor: map[string]any{
			"offset": 0,
		},
	}, nil
}

func (p *cursorTestProvider) LoadCursorBatch(_ context.Context, _ ExecutionContext, req CursorBatchRequest) (CursorBatchResult, error) {
	p.loadCalls++
	offset := cursorTestOffset(req.Cursor)
	if offset >= len(p.batches) {
		return CursorBatchResult{
			Batch:      Batch{Items: []json.RawMessage{}},
			NextCursor: map[string]any{"offset": offset},
			HasMore:    false,
		}, nil
	}

	nextOffset := offset + 1
	return CursorBatchResult{
		Batch:      p.batches[offset],
		NextCursor: map[string]any{"offset": nextOffset},
		HasMore:    nextOffset < len(p.batches),
		Metadata: map[string]any{
			"offset": offset,
		},
	}, nil
}

type cursorTestStateStore struct {
	summary      Summary
	runtime      *strictRuntimeValues
	ready        bool
	totalShards  int
	completed    map[int]bool
	registered   bool
	completedCnt int64
}

func newCursorTestStateStore() *cursorTestStateStore {
	return &cursorTestStateStore{
		runtime:   &strictRuntimeValues{values: make(map[string]any)},
		completed: make(map[int]bool),
	}
}

type strictRuntimeValues struct {
	values map[string]any
}

func (r *strictRuntimeValues) Set(_ context.Context, key string, value any) error {
	r.values[key] = value
	return nil
}

func (r *strictRuntimeValues) Get(_ context.Context, key string, dest any) error {
	value, ok := r.values[key]
	if !ok {
		return fmt.Errorf("runtime key not found: %s", key)
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(payload, dest)
}

func (r *strictRuntimeValues) Delete(_ context.Context, key string) error {
	delete(r.values, key)
	return nil
}

func (s *cursorTestStateStore) RuntimeValues(Input, time.Duration) RuntimeValues {
	return s.runtime
}

func (s *cursorTestStateStore) Initialize(_ context.Context, _ Input, _ []Batch, summary Summary, _ map[string]any, _ time.Duration) (string, error) {
	s.summary = summary
	s.ready = true
	return "cursor:test:batches", nil
}

func (s *cursorTestStateStore) LoadSummary(context.Context, Input) (Summary, error) {
	if !s.ready {
		return Summary{}, assert.AnError
	}
	return s.summary, nil
}

func (s *cursorTestStateStore) LoadBatch(context.Context, string, int) (Batch, error) {
	return Batch{}, fmt.Errorf("LoadBatch no esperado en source_mode=cursor")
}

func (s *cursorTestStateStore) Cleanup(context.Context, Input, string) error {
	return nil
}

func (s *cursorTestStateStore) SetCounter(context.Context, string, int64, time.Duration) error {
	return nil
}

func (s *cursorTestStateStore) IncrCounter(context.Context, string, int64, time.Duration) (int64, error) {
	return 0, nil
}

func (s *cursorTestStateStore) GetCounter(context.Context, string) (int64, error) {
	return 0, nil
}

func (s *cursorTestStateStore) RegisterShards(_ context.Context, _ Input, totalShards int, _ time.Duration) error {
	s.totalShards = totalShards
	s.registered = true
	return nil
}

func (s *cursorTestStateStore) CompleteShard(_ context.Context, _ Input, shardIndex int, totalShards int, _ time.Duration) (ShardCompletion, error) {
	if !s.completed[shardIndex] {
		s.completed[shardIndex] = true
		s.completedCnt++
	}
	if s.totalShards == 0 {
		s.totalShards = totalShards
	}
	return ShardCompletion{
		CompletedShards: s.completedCnt,
		ShouldFinalize:  s.completedCnt == int64(s.totalShards),
	}, nil
}

func TestManagerStartAndDispatchShards_CursorModeForcesSingleShard(t *testing.T) {
	provider := &cursorTestProvider{
		batches: []Batch{
			{Items: buildTestItems(2)},
			{Items: buildTestItems(1)},
		},
	}
	stateStore := newCursorTestStateStore()
	manager := NewManager(fakeLifecycleWithoutProgress{}, provider, &fakeBatchProcessor{}, nil, stateStore, time.Minute, nil)

	startRes, err := manager.Start(context.Background(), StartRequest{
		Input: Input{
			ParentID: 1,
			RedisKey: "cursor-run-start",
		},
		BatchSize:  2,
		RedisTTL:   time.Minute,
		SourceMode: SourceModeCursor,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, startRes.TotalBatches)
	assert.Equal(t, 1, provider.prepareCalls)
	assert.Equal(t, "cursor:test:batches", startRes.BatchesListKey)
	assert.Equal(t, SourceModeCursor, startRes.Metadata["source_mode"])

	var runtimeSourceMode string
	require.NoError(t, stateStore.runtime.Get(context.Background(), "source_mode", &runtimeSourceMode))
	assert.Equal(t, SourceModeCursor, runtimeSourceMode)

	var runtimeBatchSize int
	require.NoError(t, stateStore.runtime.Get(context.Background(), "cursor_batch_size", &runtimeBatchSize))
	assert.Equal(t, 2, runtimeBatchSize)

	dispatchRes, err := manager.DispatchShards(context.Background(), DispatchRequest{
		Input: Input{
			ParentID: 1,
			RedisKey: "cursor-run-start",
		},
		TotalBatches:   startRes.TotalBatches,
		ParallelShards: 4,
		SourceMode:     SourceModeCursor,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, dispatchRes.TotalShards)
	assert.Equal(t, []int{0}, dispatchRes.InitialBatchIndexes)
	assert.True(t, stateStore.registered)
}

func TestManagerProcessBatch_CursorModeAdvancesUntilShardCompletion(t *testing.T) {
	provider := &cursorTestProvider{
		batches: []Batch{
			{Items: buildTestItems(2)},
			{Items: buildTestItems(1)},
		},
	}
	stateStore := newCursorTestStateStore()
	lifecycle := &fakeLifecycleWithProgress{}
	processor := &fakeBatchProcessor{}
	manager := NewManager(lifecycle, provider, processor, nil, stateStore, time.Minute, nil)

	_, err := manager.Start(context.Background(), StartRequest{
		Input: Input{
			ParentID: 1,
			RedisKey: "cursor-run-process",
		},
		BatchSize:  2,
		RedisTTL:   time.Minute,
		SourceMode: SourceModeCursor,
	})
	require.NoError(t, err)
	require.NoError(t, stateStore.RegisterShards(context.Background(), Input{ParentID: 1, RedisKey: "cursor-run-process"}, 1, time.Minute))

	first, err := manager.ProcessBatch(context.Background(), ProcessRequest{
		Input: Input{
			ParentID: 1,
			RedisKey: "cursor-run-process",
		},
		BatchIndex:        0,
		TotalBatches:      1,
		ConcurrentBatches: 1,
		ShardIndex:        0,
		TotalShards:       1,
		SourceMode:        SourceModeCursor,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, first.NextBatchIndex)
	assert.False(t, first.IsShardComplete)
	assert.False(t, first.IsLastBatch)
	assert.Equal(t, 2, first.ProcessedCount)
	assert.Equal(t, 1, first.BatchesProcessed)

	second, err := manager.ProcessBatch(context.Background(), ProcessRequest{
		Input: Input{
			ParentID: 1,
			RedisKey: "cursor-run-process",
		},
		BatchIndex:        first.NextBatchIndex,
		TotalBatches:      1,
		ConcurrentBatches: 1,
		ShardIndex:        0,
		TotalShards:       1,
		SourceMode:        SourceModeCursor,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, second.NextBatchIndex)
	assert.True(t, second.IsShardComplete)
	assert.True(t, second.IsLastBatch)
	assert.True(t, second.ShouldDispatchNextStep)
	assert.Equal(t, int64(1), second.CompletedShards)
	assert.Equal(t, 1, second.ProcessedCount)

	assert.Equal(t, 2, provider.loadCalls)
	assert.Equal(t, []int{2, 1}, processor.sizes)
	assert.Equal(t, []int{2, 1}, lifecycle.sizes)

	var currentCursor map[string]any
	require.NoError(t, stateStore.runtime.Get(context.Background(), cursorStateRuntimeKey(0), &currentCursor))
	assert.Equal(t, 2, cursorTestOffset(currentCursor))
}

func TestManagerProcessBatch_CursorModeWithDispatchPacingReusesPendingBatch(t *testing.T) {
	provider := &cursorTestProvider{
		batches: []Batch{
			{Items: buildTestItems(5)},
		},
	}
	stateStore := newCursorTestStateStore()
	lifecycle := &fakeLifecycleWithProgress{}
	processor := &fakeBatchProcessor{}
	manager := NewManager(lifecycle, provider, processor, nil, stateStore, time.Minute, nil)

	_, err := manager.Start(context.Background(), StartRequest{
		Input: Input{
			ParentID: 1,
			RedisKey: "cursor-run-pacing",
		},
		BatchSize:  5,
		RedisTTL:   time.Minute,
		SourceMode: SourceModeCursor,
	})
	require.NoError(t, err)
	require.NoError(t, stateStore.RegisterShards(context.Background(), Input{ParentID: 1, RedisKey: "cursor-run-pacing"}, 1, time.Minute))

	req := ProcessRequest{
		Input: Input{
			ParentID: 1,
			RedisKey: "cursor-run-pacing",
		},
		BatchIndex:        0,
		TotalBatches:      1,
		ConcurrentBatches: 1,
		ShardIndex:        0,
		TotalShards:       1,
		SourceMode:        SourceModeCursor,
		DispatchPacing: DispatchPacingConfig{
			Enabled:             true,
			MessagesPerInterval: 2,
			IntervalSeconds:     2,
		},
	}

	first, err := manager.ProcessBatch(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, 0, first.NextBatchIndex)
	assert.False(t, first.IsShardComplete)
	assert.Equal(t, 2, first.ProcessedCount)
	assert.Equal(t, 1, provider.loadCalls)

	second, err := manager.ProcessBatch(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, 0, second.NextBatchIndex)
	assert.False(t, second.IsShardComplete)
	assert.Equal(t, 2, second.ProcessedCount)
	assert.Equal(t, 1, provider.loadCalls)

	third, err := manager.ProcessBatch(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, 1, third.NextBatchIndex)
	assert.True(t, third.IsShardComplete)
	assert.True(t, third.IsLastBatch)
	assert.True(t, third.ShouldDispatchNextStep)
	assert.Equal(t, 1, third.ProcessedCount)
	assert.Equal(t, 1, provider.loadCalls)
	assert.Equal(t, []int{2, 2, 1}, processor.sizes)
	assert.Equal(t, []int{2, 2, 1}, lifecycle.sizes)

	var pending Batch
	err = stateStore.runtime.Get(context.Background(), pendingCursorBatchRuntimeKey(0, 0), &pending)
	require.Error(t, err)
}

func cursorTestOffset(cursor map[string]any) int {
	if len(cursor) == 0 {
		return 0
	}
	switch value := cursor["offset"].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}
