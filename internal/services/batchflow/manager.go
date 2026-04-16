package batchflow

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type manager struct {
	lifecycle     ParentLifecycle
	dataProvider  DataProvider
	processor     BatchProcessor
	finalizer     Finalizer
	stateStore    StateStore
	defaultTTL    time.Duration
}

func NewManager(
	lifecycle ParentLifecycle,
	dataProvider DataProvider,
	processor BatchProcessor,
	finalizer Finalizer,
	stateStore StateStore,
	defaultTTL time.Duration,
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

func (m *manager) ProcessBatch(ctx context.Context, req ProcessRequest) (ProcessResult, error) {
	if err := validateInput(req.Input); err != nil {
		return ProcessResult{}, err
	}
	if req.BatchIndex < 0 || req.BatchIndex >= req.TotalBatches {
		return ProcessResult{}, fmt.Errorf("batch_index fuera de rango: %d (total=%d)", req.BatchIndex, req.TotalBatches)
	}
	if req.ConcurrentBatches <= 0 {
		req.ConcurrentBatches = 1
	}
	if req.ConcurrentBatches > req.TotalBatches-req.BatchIndex {
		req.ConcurrentBatches = req.TotalBatches - req.BatchIndex
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

	results := make(chan batchResult, req.ConcurrentBatches)
	var wg sync.WaitGroup
	for offset := 0; offset < req.ConcurrentBatches; offset++ {
		batchIndex := req.BatchIndex + offset
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

	nextIndex := req.BatchIndex + req.ConcurrentBatches
	isLast := nextIndex >= req.TotalBatches

	return ProcessResult{
		NextBatchIndex:  nextIndex,
		IsLastBatch:     isLast,
		ProcessedCount:  totalProcessed,
		BatchesProcessed: req.ConcurrentBatches,
		Metadata:        metadata,
	}, nil
}

func (m *manager) Finalize(ctx context.Context, req FinalizeRequest) (FinalizeResult, error) {
	if err := validateInput(req.Input); err != nil {
		return FinalizeResult{}, err
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
	return result, nil
}

func (m *manager) Fail(ctx context.Context, input Input, cause error) error {
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
