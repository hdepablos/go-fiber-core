package batchflow

import (
	"context"
	"testing"
)

type stubManager struct{}

func (stubManager) Start(context.Context, StartRequest) (StartResult, error) {
	return StartResult{}, nil
}
func (stubManager) DispatchShards(context.Context, DispatchRequest) (DispatchResult, error) {
	return DispatchResult{}, nil
}
func (stubManager) ProcessBatch(context.Context, ProcessRequest) (ProcessResult, error) {
	return ProcessResult{}, nil
}
func (stubManager) Finalize(context.Context, FinalizeRequest) (FinalizeResult, error) {
	return FinalizeResult{}, nil
}
func (stubManager) Fail(context.Context, Input, error) error { return nil }

func TestManagerRegistryResolveByExecutionKey(t *testing.T) {
	registry := NewManagerRegistry()
	expected := stubManager{}

	registry.Register(func(ctx context.Context) (Manager, error) {
		return expected, nil
	}, "bulk/process/test/process_batch")

	got, err := registry.Resolve(context.Background(), "bulk/process/test/process_batch")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if got == nil {
		t.Fatalf("Resolve returned nil manager")
	}
}
