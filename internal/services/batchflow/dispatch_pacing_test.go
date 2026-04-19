package batchflow

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePreviewProvider struct {
	components PreviewComponents
}

func (p fakePreviewProvider) PreviewComponents() PreviewComponents {
	return p.components
}

type fakeDataProvider struct {
	load LoadBatchesResult
}

func (p fakeDataProvider) LoadBatches(context.Context, ExecutionContext, int) (LoadBatchesResult, error) {
	return p.load, nil
}

type fakeBatchPreviewer struct{}

func (fakeBatchPreviewer) PreviewBatch(_ context.Context, _ ExecutionContext, batch Batch) (PreviewBatchResult, error) {
	items := make([]PreviewItemResult, 0, len(batch.Items))
	for idx := range batch.Items {
		items = append(items, PreviewItemResult{
			RowNumber: idx + 1,
			Status:    "previewed",
		})
	}
	return PreviewBatchResult{
		Items:          items,
		ProcessedCount: len(items),
	}, nil
}

type fakeBatchProcessor struct {
	sizes []int
}

func (p *fakeBatchProcessor) ProcessBatch(_ context.Context, _ ExecutionContext, batch Batch) (ProcessBatchResult, error) {
	p.sizes = append(p.sizes, len(batch.Items))
	return ProcessBatchResult{
		ProcessedCount: len(batch.Items),
		Metadata: map[string]any{
			"size": len(batch.Items),
		},
	}, nil
}

type fakeStateStore struct {
	summary  Summary
	batches  []Batch
	counters map[string]int64
	runtime  *fakeRuntimeValues
	ready    bool
}

func newFakeStateStore() *fakeStateStore {
	return &fakeStateStore{
		counters: make(map[string]int64),
		runtime:  &fakeRuntimeValues{values: make(map[string]any)},
	}
}

func (s *fakeStateStore) RuntimeValues(Input, time.Duration) RuntimeValues {
	return s.runtime
}

func (s *fakeStateStore) Initialize(_ context.Context, _ Input, batches []Batch, summary Summary, _ map[string]any, _ time.Duration) (string, error) {
	s.batches = append([]Batch(nil), batches...)
	s.summary = summary
	s.ready = true
	return "preview:batches", nil
}

func (s *fakeStateStore) LoadSummary(context.Context, Input) (Summary, error) {
	if !s.ready {
		return Summary{}, assert.AnError
	}
	return s.summary, nil
}

func (s *fakeStateStore) LoadBatch(_ context.Context, _ string, batchIndex int) (Batch, error) {
	return s.batches[batchIndex], nil
}

func (s *fakeStateStore) Cleanup(context.Context, Input, string) error {
	return nil
}

func (s *fakeStateStore) SetCounter(_ context.Context, key string, value int64, _ time.Duration) error {
	s.counters[key] = value
	return nil
}

func (s *fakeStateStore) IncrCounter(_ context.Context, key string, delta int64, _ time.Duration) (int64, error) {
	s.counters[key] += delta
	return s.counters[key], nil
}

func (s *fakeStateStore) GetCounter(_ context.Context, key string) (int64, error) {
	return s.counters[key], nil
}

func (s *fakeStateStore) RegisterShards(context.Context, Input, int, time.Duration) error {
	return nil
}

func (s *fakeStateStore) CompleteShard(context.Context, Input, int, int, time.Duration) (ShardCompletion, error) {
	return ShardCompletion{}, nil
}

type fakeRuntimeValues struct {
	values map[string]any
}

func (r *fakeRuntimeValues) Set(_ context.Context, key string, value any) error {
	r.values[key] = value
	return nil
}

func (r *fakeRuntimeValues) Get(_ context.Context, key string, dest any) error {
	payload, err := json.Marshal(r.values[key])
	if err != nil {
		return err
	}
	return json.Unmarshal(payload, dest)
}

func (r *fakeRuntimeValues) Delete(_ context.Context, key string) error {
	delete(r.values, key)
	return nil
}

func TestResolveDispatchPacingConfig_DefaultDisabledWhenMissing(t *testing.T) {
	cfg, err := ResolveDispatchPacingConfig(nil)
	require.NoError(t, err)
	assert.False(t, cfg.Enabled)
	assert.Zero(t, cfg.MessagesPerInterval)
	assert.Zero(t, cfg.IntervalSeconds)
}

func TestPreviewApplyChanges_UsesDispatchPacing(t *testing.T) {
	processor := &fakeBatchProcessor{}
	stateStore := newFakeStateStore()
	registry := NewPreviewRegistry()
	registry.Register("test pacing", func(context.Context) (PreviewProvider, error) {
		return fakePreviewProvider{
			components: PreviewComponents{
				DataProvider: fakeDataProvider{
					load: LoadBatchesResult{
						Batches: []Batch{{Items: buildTestItems(10)}},
						Summary: Summary{TotalRecords: 10},
					},
				},
				BatchProcessor: processor,
				BatchPreviewer: fakeBatchPreviewer{},
				StateStore:     stateStore,
			},
		}, nil
	})

	originalNow := dispatchPacingNowFn
	originalSleep := dispatchPacingSleepFn
	currentTime := time.Unix(1_700_000_000, 0)
	dispatchPacingNowFn = func() time.Time { return currentTime }
	dispatchPacingSleepFn = func(_ context.Context, d time.Duration) error {
		currentTime = currentTime.Add(d)
		return nil
	}
	defer func() {
		dispatchPacingNowFn = originalNow
		dispatchPacingSleepFn = originalSleep
	}()

	svc := NewPreviewService(registry, time.Minute)
	res, err := svc.Preview(context.Background(), PreviewRequest{
		ProcessTypeID:   1,
		ProcessTypeName: "test pacing",
		Mode:            "all",
		ApplyChanges:    true,
		DispatchPacing: DispatchPacingConfig{
			Enabled:             true,
			MessagesPerInterval: 3,
			IntervalSeconds:     5,
		},
		Input: Input{
			ParentID: 1,
			RedisKey: "preview-pacing",
		},
		Limit: 10,
	})
	require.NoError(t, err)
	assert.True(t, res.AppliedChanges)
	assert.Equal(t, []int{3, 3, 3, 1}, processor.sizes)

	meta, ok := res.ApplyChangesMetadata["dispatch_pacing"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 4, meta["chunk_count"])
	assert.Equal(t, 3, meta["messages_per_interval"])
	assert.Equal(t, int64(5), meta["interval_seconds"])
}

func buildTestItems(total int) []json.RawMessage {
	items := make([]json.RawMessage, 0, total)
	for i := 1; i <= total; i++ {
		items = append(items, json.RawMessage([]byte(`{"id":`+strconv.Itoa(i)+`}`)))
	}
	return items
}
