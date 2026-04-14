package exportmanager

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

type fakePreviewProvider struct {
	components PreviewComponents
}

func (f fakePreviewProvider) PreviewComponents() PreviewComponents {
	return f.components
}

type countingDataProvider struct {
	calls int
}

func (p *countingDataProvider) LoadBatches(_ context.Context, execCtx ExecutionContext, _ int) (LoadBatchesResult, error) {
	p.calls++
	batches := []Batch{
		{Items: []json.RawMessage{json.RawMessage(`"row-1"`), json.RawMessage(`"row-2"`)}},
		{Items: []json.RawMessage{json.RawMessage(`"row-3"`)}},
	}
	return LoadBatchesResult{
		Batches: batches,
		Summary: Summary{
			TotalRecords: 3,
			TotalAmount:  15.5,
		},
	}, nil
}

type fakeHeaderBuilder struct{}

func (f fakeHeaderBuilder) BuildHeader(_ context.Context, execCtx ExecutionContext) ([]string, error) {
	return []string{"header:" + itoa(int(execCtx.Summary.TotalRecords))}, nil
}

type fakeBodyBuilder struct{}

func (f fakeBodyBuilder) BuildBodyLines(_ context.Context, _ ExecutionContext, item json.RawMessage) ([]string, error) {
	var value string
	if err := json.Unmarshal(item, &value); err != nil {
		return nil, err
	}
	return []string{value}, nil
}

type fakeFooterBuilder struct{}

func (f fakeFooterBuilder) BuildFooter(_ context.Context, execCtx ExecutionContext) ([]string, error) {
	return []string{"footer:" + itoa(int(execCtx.Summary.TotalRecords))}, nil
}

type memoryStateStore struct {
	summaries map[string]Summary
	batches   map[string][]Batch
	runtime   map[string]map[string][]byte
}

func newMemoryStateStore() *memoryStateStore {
	return &memoryStateStore{
		summaries: make(map[string]Summary),
		batches:   make(map[string][]Batch),
		runtime:   make(map[string]map[string][]byte),
	}
}

func (s *memoryStateStore) Initialize(_ context.Context, input Input, batches []Batch, summary Summary, _ map[string]any, _ time.Duration) (string, string, error) {
	s.summaries[input.RedisKey] = summary
	s.batches[input.RedisKey] = batches
	return input.RedisKey + ":batches", input.RedisKey + ":parts", nil
}

func (s *memoryStateStore) LoadSummary(_ context.Context, input Input) (Summary, error) {
	summary, ok := s.summaries[input.RedisKey]
	if !ok {
		return Summary{}, context.Canceled
	}
	return summary, nil
}

func (s *memoryStateStore) LoadBatch(_ context.Context, batchesListKey string, batchIndex int) (Batch, error) {
	redisKey := batchesListKey[:len(batchesListKey)-len(":batches")]
	return s.batches[redisKey][batchIndex], nil
}

func (s *memoryStateStore) AppendPartKey(context.Context, string, string) error {
	return nil
}

func (s *memoryStateStore) LoadPartKeys(context.Context, string) ([]string, error) {
	return nil, nil
}

func (s *memoryStateStore) Cleanup(context.Context, Input, string, string) error {
	return nil
}

func (s *memoryStateStore) RuntimeValues(input Input, _ time.Duration) RuntimeValues {
	if _, ok := s.runtime[input.RedisKey]; !ok {
		s.runtime[input.RedisKey] = make(map[string][]byte)
	}
	return &memoryRuntime{store: s.runtime[input.RedisKey]}
}

type memoryRuntime struct {
	store map[string][]byte
}

func (r *memoryRuntime) Set(_ context.Context, key string, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	r.store[key] = payload
	return nil
}

func (r *memoryRuntime) Get(_ context.Context, key string, dest any) error {
	return json.Unmarshal(r.store[key], dest)
}

func (r *memoryRuntime) Delete(_ context.Context, key string) error {
	delete(r.store, key)
	return nil
}

func TestNormalizeFiltersSupportsArrayContract(t *testing.T) {
	raw := []any{
		map[string]any{"field": "status_code", "operator": "eq", "value": "ERROR_PROCESS"},
		map[string]any{"field": "row_number", "operator": "in", "value": []any{10, 11}},
	}

	normalized, err := NormalizeFilters(raw)
	if err != nil {
		t.Fatalf("NormalizeFilters() error = %v", err)
	}

	if got := normalized["status_code"]; got != "ERROR_PROCESS" {
		t.Fatalf("status_code = %v, want ERROR_PROCESS", got)
	}
	values, ok := normalized["row_number"].([]any)
	if !ok || len(values) != 2 {
		t.Fatalf("row_number = %#v, want 2 values", normalized["row_number"])
	}
}

func TestPreviewServiceReusesPreparedSession(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStateStore()
	dataProvider := &countingDataProvider{}
	registry := NewPreviewRegistry()
	registry.Register("demo export", func(context.Context) (PreviewProvider, error) {
		return fakePreviewProvider{
			components: PreviewComponents{
				DataProvider:  dataProvider,
				HeaderBuilder: fakeHeaderBuilder{},
				BodyBuilder:   fakeBodyBuilder{},
				FooterBuilder: fakeFooterBuilder{},
				StateStore:    store,
			},
		}, nil
	}, "demo/export/start")

	service := NewPreviewService(registry, time.Minute)

	prepareResp, err := service.Preview(ctx, PreviewRequest{
		ProcessTypeID: 21,
		ProcessTypeName: "demo export",
		ExecutionKeys: []string{"demo/export/start"},
		Mode:          "prepare",
		Input: Input{
			ParentID: 1,
			RedisKey: "session-1",
		},
	})
	if err != nil {
		t.Fatalf("prepare preview error = %v", err)
	}
	if prepareResp.TotalBatches != 2 {
		t.Fatalf("prepare TotalBatches = %d, want 2", prepareResp.TotalBatches)
	}
	if dataProvider.calls != 1 {
		t.Fatalf("LoadBatches calls = %d, want 1", dataProvider.calls)
	}

	headerResp, err := service.Preview(ctx, PreviewRequest{
		ProcessTypeID: 21,
		ProcessTypeName: "demo export",
		ExecutionKeys: []string{"demo/export/start"},
		Mode:          "header",
		Input: Input{
			ParentID: 1,
			RedisKey: "session-1",
		},
	})
	if err != nil {
		t.Fatalf("header preview error = %v", err)
	}
	if dataProvider.calls != 1 {
		t.Fatalf("LoadBatches calls after header = %d, want 1", dataProvider.calls)
	}
	if len(headerResp.HeaderLines) != 1 || headerResp.HeaderLines[0] != "header:3" {
		t.Fatalf("header lines = %#v, want header:3", headerResp.HeaderLines)
	}
}

func TestPreviewServiceRendersSelectedBodyWindow(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStateStore()
	dataProvider := &countingDataProvider{}
	registry := NewPreviewRegistry()
	registry.Register("demo export", func(context.Context) (PreviewProvider, error) {
		return fakePreviewProvider{
			components: PreviewComponents{
				DataProvider:  dataProvider,
				HeaderBuilder: fakeHeaderBuilder{},
				BodyBuilder:   fakeBodyBuilder{},
				FooterBuilder: fakeFooterBuilder{},
				StateStore:    store,
			},
		}, nil
	}, "demo/export/start")

	service := NewPreviewService(registry, time.Minute)

	resp, err := service.Preview(ctx, PreviewRequest{
		ProcessTypeID:   21,
		ProcessTypeName: "demo export",
		ExecutionKeys:   []string{"demo/export/start"},
		Mode:            "all",
		Limit:           2,
		Offset:          1,
		Input: Input{
			ParentID: 1,
			RedisKey: "session-2",
		},
	})
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}

	if got, want := resp.BodyLines, []string{"row-2", "row-3"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("body lines = %#v, want %#v", got, want)
	}
	if len(resp.HeaderLines) != 1 || len(resp.FooterLines) != 1 {
		t.Fatalf("expected header and footer lines, got %#v / %#v", resp.HeaderLines, resp.FooterLines)
	}
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	out := make([]byte, 0, 12)
	for value > 0 {
		out = append([]byte{byte('0' + value%10)}, out...)
		value /= 10
	}
	return string(out)
}
