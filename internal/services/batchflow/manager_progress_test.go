package batchflow

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeLifecycleWithProgress struct {
	sizes []int
}

func (l *fakeLifecycleWithProgress) Start(context.Context, ExecutionContext) error {
	return nil
}

func (l *fakeLifecycleWithProgress) End(context.Context, ExecutionContext, FinalizeResult) error {
	return nil
}

func (l *fakeLifecycleWithProgress) Fail(context.Context, ExecutionContext, error) error {
	return nil
}

func (l *fakeLifecycleWithProgress) RefreshProgress(_ context.Context, _ ExecutionContext, batch Batch) error {
	l.sizes = append(l.sizes, len(batch.Items))
	return nil
}

type fakeLifecycleWithoutProgress struct{}

func (fakeLifecycleWithoutProgress) Start(context.Context, ExecutionContext) error {
	return nil
}

func (fakeLifecycleWithoutProgress) End(context.Context, ExecutionContext, FinalizeResult) error {
	return nil
}

func (fakeLifecycleWithoutProgress) Fail(context.Context, ExecutionContext, error) error {
	return nil
}

func TestManagerProcessBatch_RefreshesProgressPerProcessedBatch(t *testing.T) {
	stateStore := newFakeStateStore()
	stateStore.summary = Summary{TotalRecords: 3}
	stateStore.batches = []Batch{
		{Items: buildTestItems(2)},
		{Items: buildTestItems(1)},
	}
	stateStore.ready = true

	processor := &fakeBatchProcessor{}
	lifecycle := &fakeLifecycleWithProgress{}
	manager := NewManager(lifecycle, fakeDataProvider{}, processor, nil, stateStore, time.Minute, nil)

	res, err := manager.ProcessBatch(context.Background(), ProcessRequest{
		Input: Input{
			ParentID: 1,
			RedisKey: "run-progress",
		},
		BatchesListKey:    "preview:batches",
		BatchIndex:        0,
		TotalBatches:      2,
		ConcurrentBatches: 2,
		ShardIndex:        0,
		TotalShards:       1,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, res.ProcessedCount)
	assert.ElementsMatch(t, []int{2, 1}, processor.sizes)
	assert.ElementsMatch(t, []int{2, 1}, lifecycle.sizes)
}

func TestManagerProcessBatch_AllowsLifecycleWithoutOptionalProgressHook(t *testing.T) {
	stateStore := newFakeStateStore()
	stateStore.summary = Summary{TotalRecords: 2}
	stateStore.batches = []Batch{
		{Items: buildTestItems(2)},
	}
	stateStore.ready = true

	processor := &fakeBatchProcessor{}
	manager := NewManager(fakeLifecycleWithoutProgress{}, fakeDataProvider{}, processor, nil, stateStore, time.Minute, nil)

	res, err := manager.ProcessBatch(context.Background(), ProcessRequest{
		Input: Input{
			ParentID: 1,
			RedisKey: "run-no-progress-hook",
		},
		BatchesListKey:    "preview:batches",
		BatchIndex:        0,
		TotalBatches:      1,
		ConcurrentBatches: 1,
		ShardIndex:        0,
		TotalShards:       1,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, res.ProcessedCount)
	assert.Equal(t, []int{2}, processor.sizes)
}
