package batchflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	servicecontracts "go-fiber-core/internal/services/serviceconfig/contracts"
)

type DispatchPacingConfig struct {
	Enabled             bool  `json:"enabled"`
	MessagesPerInterval int   `json:"messages_per_interval"`
	IntervalSeconds     int64 `json:"interval_seconds"`
}

const (
	minDispatchPacingIntervalSeconds = 1
	maxDispatchPacingIntervalSeconds = 10
)

func ResolveDispatchPacingConfig(stepConfig map[string]any) (DispatchPacingConfig, error) {
	if len(stepConfig) == 0 {
		return DispatchPacingConfig{}, nil
	}
	raw, ok := stepConfig["dispatch_pacing"]
	if !ok || raw == nil {
		return DispatchPacingConfig{}, nil
	}
	return ParseDispatchPacingConfig(raw)
}

func ParseDispatchPacingConfig(raw any) (DispatchPacingConfig, error) {
	if raw == nil {
		return DispatchPacingConfig{}, nil
	}

	payload, err := json.Marshal(raw)
	if err != nil {
		return DispatchPacingConfig{}, fmt.Errorf("dispatch_pacing inválido: %w", err)
	}

	var cfg DispatchPacingConfig
	if err := json.Unmarshal(payload, &cfg); err != nil {
		return DispatchPacingConfig{}, fmt.Errorf("dispatch_pacing inválido: %w", err)
	}
	if !cfg.Enabled {
		return DispatchPacingConfig{}, nil
	}
	if cfg.MessagesPerInterval <= 0 {
		return DispatchPacingConfig{}, fmt.Errorf("dispatch_pacing.messages_per_interval debe ser mayor a 0")
	}
	if cfg.IntervalSeconds < minDispatchPacingIntervalSeconds || cfg.IntervalSeconds > maxDispatchPacingIntervalSeconds {
		return DispatchPacingConfig{}, fmt.Errorf("dispatch_pacing.interval_seconds debe estar entre %d y %d", minDispatchPacingIntervalSeconds, maxDispatchPacingIntervalSeconds)
	}
	return cfg, nil
}

func ValidateDispatchPacingStepConfig(stepConfig map[string]any) (DispatchPacingConfig, error) {
	cfg, err := ResolveDispatchPacingConfig(stepConfig)
	if err != nil || !cfg.Enabled {
		return cfg, err
	}

	rawPolicy, ok := stepConfig["execution_policy"]
	if !ok || rawPolicy == nil {
		return DispatchPacingConfig{}, fmt.Errorf("dispatch_pacing requiere execution_policy configurado")
	}

	policyBytes, err := json.Marshal(rawPolicy)
	if err != nil {
		return DispatchPacingConfig{}, fmt.Errorf("execution_policy inválido para dispatch_pacing: %w", err)
	}

	var policy servicecontracts.ExecutionPolicy
	if err := json.Unmarshal(policyBytes, &policy); err != nil {
		return DispatchPacingConfig{}, fmt.Errorf("execution_policy inválido para dispatch_pacing: %w", err)
	}

	if !strings.EqualFold(policy.Mode, "ASYNC") {
		return DispatchPacingConfig{}, fmt.Errorf("dispatch_pacing requiere execution_policy.mode=ASYNC")
	}
	if !policy.AutoInvoke.Enabled {
		return DispatchPacingConfig{}, fmt.Errorf("dispatch_pacing requiere execution_policy.auto_invoke.enabled=true")
	}
	if policy.AutoInvoke.DelaySeconds > 0 && int64(policy.AutoInvoke.DelaySeconds) != cfg.IntervalSeconds {
		return DispatchPacingConfig{}, fmt.Errorf("execution_policy.auto_invoke.delay_seconds debe coincidir con dispatch_pacing.interval_seconds")
	}

	return cfg, nil
}

type DispatchPacingInvocationResult struct {
	ProcessResult  ProcessBatchResult
	NextBatchIndex int
	BatchComplete  bool
}

func ProcessBatchWithDispatchPacing(
	ctx context.Context,
	processor BatchProcessor,
	execCtx ExecutionContext,
	batch Batch,
	batchIndex int,
	cfg DispatchPacingConfig,
	totalShards int,
) (DispatchPacingInvocationResult, error) {
	if processor == nil {
		return DispatchPacingInvocationResult{}, fmt.Errorf("batch processor inválido")
	}
	if !cfg.Enabled || len(batch.Items) == 0 {
		res, err := processor.ProcessBatch(ctx, execCtx, batch)
		if err != nil {
			return DispatchPacingInvocationResult{}, err
		}
		return DispatchPacingInvocationResult{
			ProcessResult:  res,
			NextBatchIndex: batchIndex + max(totalShards, 1),
			BatchComplete:  true,
		}, nil
	}

	offset, err := loadDispatchPacingOffset(ctx, execCtx.Runtime, batchIndex)
	if err != nil {
		return DispatchPacingInvocationResult{}, err
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= len(batch.Items) {
		offset = 0
	}

	end := offset + cfg.MessagesPerInterval
	if end > len(batch.Items) {
		end = len(batch.Items)
	}
	res, err := processor.ProcessBatch(ctx, execCtx, Batch{Items: batch.Items[offset:end]})
	if err != nil {
		return DispatchPacingInvocationResult{}, err
	}

	batchComplete := end >= len(batch.Items)
	nextBatchIndex := batchIndex
	if batchComplete {
		if err := clearDispatchPacingOffset(ctx, execCtx.Runtime, batchIndex); err != nil {
			return DispatchPacingInvocationResult{}, err
		}
		nextBatchIndex = batchIndex + max(totalShards, 1)
	} else {
		if err := saveDispatchPacingOffset(ctx, execCtx.Runtime, batchIndex, end); err != nil {
			return DispatchPacingInvocationResult{}, err
		}
	}

	res.Metadata = appendDispatchPacingInvocationMetadata(res.Metadata, cfg, batchIndex, offset, end, len(batch.Items), batchComplete)
	return DispatchPacingInvocationResult{
		ProcessResult:  res,
		NextBatchIndex: nextBatchIndex,
		BatchComplete:  batchComplete,
	}, nil
}

func SimulateDispatchPacingPreview(
	ctx context.Context,
	processor BatchProcessor,
	execCtx ExecutionContext,
	batch Batch,
	cfg DispatchPacingConfig,
) (ProcessBatchResult, error) {
	if processor == nil {
		return ProcessBatchResult{}, fmt.Errorf("batch processor inválido")
	}
	if !cfg.Enabled || len(batch.Items) == 0 {
		return processor.ProcessBatch(ctx, execCtx, batch)
	}

	totalProcessed := 0
	chunkSize := cfg.MessagesPerInterval
	chunkSizes := make([]int, 0, (len(batch.Items)+chunkSize-1)/chunkSize)
	waitsMs := make([]int64, 0, len(chunkSizes))
	aggregateMetadata := make(map[string]any)

	for start := 0; start < len(batch.Items); start += chunkSize {
		end := start + chunkSize
		if end > len(batch.Items) {
			end = len(batch.Items)
		}
		res, err := processor.ProcessBatch(ctx, execCtx, Batch{Items: batch.Items[start:end]})
		if err != nil {
			return ProcessBatchResult{}, err
		}
		totalProcessed += res.ProcessedCount
		if len(res.Metadata) > 0 {
			aggregateMetadata[fmt.Sprintf("chunk_%d", len(chunkSizes)+1)] = res.Metadata
		}
		chunkSizes = append(chunkSizes, end-start)
		if len(chunkSizes) == 1 {
			waitsMs = append(waitsMs, 0)
		} else {
			waitsMs = append(waitsMs, cfg.IntervalSeconds*1000)
		}
	}

	aggregateMetadata = appendDispatchPacingPreviewMetadata(aggregateMetadata, cfg, chunkSizes, waitsMs)
	return ProcessBatchResult{
		ProcessedCount: totalProcessed,
		Metadata:       aggregateMetadata,
	}, nil
}

func appendDispatchPacingInvocationMetadata(metadata map[string]any, cfg DispatchPacingConfig, batchIndex, offsetStart, offsetEnd, totalItems int, batchComplete bool) map[string]any {
	if metadata == nil {
		metadata = make(map[string]any)
	}
	metadata["dispatch_pacing"] = map[string]any{
		"enabled":               cfg.Enabled,
		"messages_per_interval": cfg.MessagesPerInterval,
		"interval_seconds":      cfg.IntervalSeconds,
		"mode":                  "auto_invoke_delay",
		"batch_index":           batchIndex,
		"offset_start":          offsetStart,
		"offset_end":            offsetEnd,
		"remaining_items":       max(totalItems-offsetEnd, 0),
		"batch_complete":        batchComplete,
		"requeue_delay_seconds": cfg.IntervalSeconds,
	}
	return metadata
}

func appendDispatchPacingPreviewMetadata(metadata map[string]any, cfg DispatchPacingConfig, chunkSizes []int, waitsMs []int64) map[string]any {
	if metadata == nil {
		metadata = make(map[string]any)
	}
	metadata["dispatch_pacing"] = map[string]any{
		"enabled":               cfg.Enabled,
		"messages_per_interval": cfg.MessagesPerInterval,
		"interval_seconds":      cfg.IntervalSeconds,
		"mode":                  "preview_simulated",
		"simulated":             true,
		"chunk_count":           len(chunkSizes),
		"chunk_sizes":           chunkSizes,
		"waits_ms":              waitsMs,
	}
	return metadata
}

func loadDispatchPacingOffset(ctx context.Context, runtime RuntimeValues, batchIndex int) (int, error) {
	if runtime == nil {
		return 0, nil
	}
	var offset int
	if err := runtime.Get(ctx, dispatchPacingOffsetKey(batchIndex), &offset); err != nil {
		return 0, nil
	}
	return offset, nil
}

func saveDispatchPacingOffset(ctx context.Context, runtime RuntimeValues, batchIndex, offset int) error {
	if runtime == nil {
		return nil
	}
	return runtime.Set(ctx, dispatchPacingOffsetKey(batchIndex), offset)
}

func clearDispatchPacingOffset(ctx context.Context, runtime RuntimeValues, batchIndex int) error {
	if runtime == nil {
		return nil
	}
	return runtime.Delete(ctx, dispatchPacingOffsetKey(batchIndex))
}

func dispatchPacingOffsetKey(batchIndex int) string {
	return fmt.Sprintf("dispatch_pacing:batch:%06d:offset", batchIndex)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
