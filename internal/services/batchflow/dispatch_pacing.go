package batchflow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type DispatchPacingConfig struct {
	Enabled             bool  `json:"enabled"`
	MessagesPerInterval int   `json:"messages_per_interval"`
	IntervalSeconds     int64 `json:"interval_seconds"`
}

var (
	dispatchPacingNowFn   = time.Now
	dispatchPacingSleepFn = sleepWithContext
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
	if cfg.IntervalSeconds <= 0 {
		return DispatchPacingConfig{}, fmt.Errorf("dispatch_pacing.interval_seconds debe ser mayor a 0")
	}
	return cfg, nil
}

func ProcessBatchWithDispatchPacing(
	ctx context.Context,
	processor BatchProcessor,
	stateStore StateStore,
	execCtx ExecutionContext,
	batch Batch,
	cfg DispatchPacingConfig,
	ttl time.Duration,
) (ProcessBatchResult, error) {
	if processor == nil {
		return ProcessBatchResult{}, fmt.Errorf("batch processor inválido")
	}
	if !cfg.Enabled || len(batch.Items) == 0 {
		return processor.ProcessBatch(ctx, execCtx, batch)
	}

	chunkSize := cfg.MessagesPerInterval
	if chunkSize <= 0 || chunkSize >= len(batch.Items) {
		waited, slot, err := acquireDispatchPacingSlot(ctx, stateStore, execCtx.Input, cfg, ttl)
		if err != nil {
			return ProcessBatchResult{}, err
		}
		res, err := processor.ProcessBatch(ctx, execCtx, batch)
		if err != nil {
			return ProcessBatchResult{}, err
		}
		res.Metadata = appendDispatchPacingMetadata(res.Metadata, cfg, []int{len(batch.Items)}, []int64{waited.Milliseconds()}, []int64{slot})
		return res, nil
	}

	totalProcessed := 0
	var (
		aggregateMetadata map[string]any
		chunkSizes        []int
		waitsMs           []int64
		slots             []int64
	)
	for start := 0; start < len(batch.Items); start += chunkSize {
		end := start + chunkSize
		if end > len(batch.Items) {
			end = len(batch.Items)
		}
		waited, slot, err := acquireDispatchPacingSlot(ctx, stateStore, execCtx.Input, cfg, ttl)
		if err != nil {
			return ProcessBatchResult{}, err
		}
		res, err := processor.ProcessBatch(ctx, execCtx, Batch{Items: batch.Items[start:end]})
		if err != nil {
			return ProcessBatchResult{}, err
		}
		totalProcessed += res.ProcessedCount
		if len(res.Metadata) > 0 {
			if aggregateMetadata == nil {
				aggregateMetadata = make(map[string]any)
			}
			aggregateMetadata[fmt.Sprintf("chunk_%d", len(chunkSizes)+1)] = res.Metadata
		}
		chunkSizes = append(chunkSizes, end-start)
		waitsMs = append(waitsMs, waited.Milliseconds())
		slots = append(slots, slot)
	}

	aggregateMetadata = appendDispatchPacingMetadata(aggregateMetadata, cfg, chunkSizes, waitsMs, slots)
	return ProcessBatchResult{
		ProcessedCount: totalProcessed,
		Metadata:       aggregateMetadata,
	}, nil
}

func acquireDispatchPacingSlot(ctx context.Context, stateStore StateStore, input Input, cfg DispatchPacingConfig, ttl time.Duration) (time.Duration, int64, error) {
	if !cfg.Enabled {
		return 0, 0, nil
	}
	if stateStore == nil {
		return 0, 0, fmt.Errorf("state store inválido para dispatch_pacing")
	}
	intervalSeconds := cfg.IntervalSeconds
	if intervalSeconds <= 0 {
		return 0, 0, fmt.Errorf("dispatch_pacing.interval_seconds inválido")
	}

	totalWait := time.Duration(0)
	for {
		now := dispatchPacingNowFn()
		slot := now.Unix() / intervalSeconds
		counterKey := fmt.Sprintf("%s:dispatch_pacing:%d", input.RedisKey, slot)
		ttlForCounter := ttl
		if ttlForCounter <= 0 {
			ttlForCounter = time.Duration(intervalSeconds*2) * time.Second
		}
		current, err := stateStore.IncrCounter(ctx, counterKey, 1, ttlForCounter)
		if err != nil {
			return totalWait, slot, err
		}
		if current == 1 {
			return totalWait, slot, nil
		}

		nextSlot := time.Unix((slot+1)*intervalSeconds, 0)
		waitFor := nextSlot.Sub(now)
		if waitFor <= 0 {
			waitFor = time.Second
		}
		if err := dispatchPacingSleepFn(ctx, waitFor); err != nil {
			return totalWait, slot, err
		}
		totalWait += waitFor
	}
}

func appendDispatchPacingMetadata(metadata map[string]any, cfg DispatchPacingConfig, chunkSizes []int, waitsMs []int64, slots []int64) map[string]any {
	if metadata == nil {
		metadata = make(map[string]any)
	}
	metadata["dispatch_pacing"] = map[string]any{
		"enabled":               cfg.Enabled,
		"messages_per_interval": cfg.MessagesPerInterval,
		"interval_seconds":      cfg.IntervalSeconds,
		"chunk_count":           len(chunkSizes),
		"chunk_sizes":           chunkSizes,
		"waits_ms":              waitsMs,
		"slots":                 slots,
	}
	return metadata
}

func sleepWithContext(ctx context.Context, waitFor time.Duration) error {
	if waitFor <= 0 {
		return nil
	}
	timer := time.NewTimer(waitFor)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
