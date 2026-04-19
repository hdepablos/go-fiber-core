package exportmanager

import (
	"context"
	"testing"
)

type stubManager struct{}

func (stubManager) Start(context.Context, StartRequest) (StartResult, error) {
	return StartResult{}, nil
}
func (stubManager) ProcessBatch(context.Context, ProcessBatchRequest) (ProcessBatchResult, error) {
	return ProcessBatchResult{}, nil
}
func (stubManager) Finalize(context.Context, FinalizeRequest) (OutputResult, error) {
	return OutputResult{}, nil
}
func (stubManager) Fail(context.Context, Input, error) error { return nil }

func TestManagerRegistryResolveByExecutionKey(t *testing.T) {
	registry := NewManagerRegistry()
	expected := stubManager{}

	registry.Register(func(ctx context.Context) (Manager, error) {
		return expected, nil
	}, "bulk/export/test/process_batch")

	got, err := registry.Resolve(context.Background(), "bulk/export/test/process_batch")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if got == nil {
		t.Fatalf("Resolve returned nil manager")
	}
}
