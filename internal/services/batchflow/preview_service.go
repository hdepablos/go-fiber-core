package batchflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type PreviewService interface {
	Preview(ctx context.Context, req PreviewRequest) (PreviewResponse, error)
}

type previewService struct {
	registry   PreviewRegistry
	sessionTTL time.Duration
}

func NewPreviewService(registry PreviewRegistry, sessionTTL time.Duration) PreviewService {
	if sessionTTL <= 0 {
		sessionTTL = 30 * time.Minute
	}
	if registry == nil {
		registry = defaultPreviewRegistry
	}
	return &previewService{
		registry:   registry,
		sessionTTL: sessionTTL,
	}
}

func (s *previewService) Preview(ctx context.Context, req PreviewRequest) (PreviewResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if req.ProcessTypeID <= 0 {
		return PreviewResponse{}, fmt.Errorf("process_type_id inválido")
	}

	provider, err := s.registry.Resolve(ctx, req.ProcessTypeName, req.ExecutionKeys)
	if err != nil {
		return PreviewResponse{}, err
	}
	components := provider.PreviewComponents()
	if components.DataProvider == nil || components.BatchPreviewer == nil || components.StateStore == nil {
		return PreviewResponse{}, fmt.Errorf("preview incompleto para process_type %q", req.ProcessTypeName)
	}
	if req.ApplyChanges && components.BatchProcessor == nil {
		return PreviewResponse{}, fmt.Errorf("preview apply_changes no disponible para process_type %q", req.ProcessTypeName)
	}

	mode := normalizePreviewMode(req.Mode)
	if req.ApplyChanges && mode == "prepare" {
		return PreviewResponse{}, fmt.Errorf("apply_changes no es compatible con mode %q", mode)
	}
	input, err := s.normalizeInput(req.ProcessTypeID, req.ResolvedProcessVersionID, req.Input, req.ItemIDs, req.RowNumbers)
	if err != nil {
		return PreviewResponse{}, err
	}

	runtimeProvider, ok := components.StateStore.(RuntimeValuesProvider)
	if !ok {
		return PreviewResponse{}, fmt.Errorf("state store no soporta runtime values")
	}

	response := PreviewResponse{
		ProcessTypeID:            req.ProcessTypeID,
		ProcessTypeName:          req.ProcessTypeName,
		ResolvedProcessVersionID: req.ResolvedProcessVersionID,
		ResolvedExecutionKeys:    append([]string(nil), req.ExecutionKeys...),
		Mode:                     mode,
		RedisKey:                 input.RedisKey,
		AppliedFilters:           input.Filters,
		BatchIndex:               req.BatchIndex,
	}

	load, totalBatches, runtimeValues, err := s.ensurePrepared(ctx, components, runtimeProvider, input, req.BatchSize)
	if err != nil {
		return PreviewResponse{}, err
	}
	response.Summary = load.Summary
	response.TotalBatches = totalBatches

	if mode == "prepare" {
		return response, nil
	}

	execCtx := ExecutionContext{
		Input:   input,
		Summary: load.Summary,
		Runtime: runtimeValues,
	}

	selectedBatch, items, selection, err := s.selectItems(ctx, components.StateStore, input, req, totalBatches)
	if err != nil {
		return PreviewResponse{}, err
	}
	response.BatchIndex = selectedBatch
	response.Selection = selection

	result, err := components.BatchPreviewer.PreviewBatch(ctx, execCtx, Batch{Items: items})
	if err != nil {
		return PreviewResponse{}, err
	}
	if req.ApplyChanges {
		processResult, err := SimulateDispatchPacingPreview(ctx, components.BatchProcessor, execCtx, Batch{Items: items}, req.DispatchPacing)
		if err != nil {
			return PreviewResponse{}, err
		}
		response.AppliedChanges = true
		response.ApplyChangesMetadata = processResult.Metadata
	}
	response.Items = result.Items
	response.RenderedCount = len(result.Items)
	return response, nil
}

func (s *previewService) normalizeInput(processTypeID int64, resolvedProcessVersionID int64, input Input, itemIDs []int64, rowNumbers []int) (Input, error) {
	if input.ParentID <= 0 {
		return Input{}, fmt.Errorf("id inválido")
	}

	filters, err := normalizeFilters(input.Filters)
	if err != nil {
		return Input{}, err
	}
	if filters == nil {
		filters = make(map[string]any)
	}
	if len(itemIDs) > 0 {
		values := make([]any, 0, len(itemIDs))
		for _, id := range itemIDs {
			values = append(values, id)
		}
		filters["id"] = values
	}
	if len(rowNumbers) > 0 {
		values := make([]any, 0, len(rowNumbers))
		for _, rowNumber := range rowNumbers {
			values = append(values, rowNumber)
		}
		filters["row_number"] = values
	}
	if len(filters) == 0 {
		filters = nil
	}
	input.Filters = filters
	input.RedisKey = composePreviewRedisKey(processTypeID, resolvedProcessVersionID, input.RedisKey)
	return input, nil
}

func (s *previewService) ensurePrepared(ctx context.Context, components PreviewComponents, runtimeProvider RuntimeValuesProvider, input Input, batchSize int) (LoadBatchesResult, int, RuntimeValues, error) {
	runtimeValues := runtimeProvider.RuntimeValues(input, s.sessionTTL)
	if summary, totalBatches, ok := s.loadPreparedSummary(ctx, components.StateStore, runtimeValues, input); ok {
		return LoadBatchesResult{Summary: summary}, totalBatches, runtimeValues, nil
	}

	if batchSize <= 0 {
		batchSize = 5000
	}

	execCtx := ExecutionContext{
		Input:   input,
		Summary: Summary{},
		Runtime: runtimeValues,
	}
	load, err := components.DataProvider.LoadBatches(ctx, execCtx, batchSize)
	if err != nil {
		return LoadBatchesResult{}, 0, nil, err
	}

	_, err = components.StateStore.Initialize(ctx, input, load.Batches, load.Summary, load.Metadata, s.sessionTTL)
	if err != nil {
		return LoadBatchesResult{}, 0, nil, err
	}

	totalBatches := len(load.Batches)
	if totalBatches == 0 {
		totalBatches = 1
	}
	_ = runtimeValues.Set(ctx, "preview_total_batches", totalBatches)
	return load, totalBatches, runtimeValues, nil
}

func (s *previewService) loadPreparedSummary(ctx context.Context, stateStore StateStore, runtimeValues RuntimeValues, input Input) (Summary, int, bool) {
	summary, err := stateStore.LoadSummary(ctx, input)
	if err != nil {
		return Summary{}, 0, false
	}
	var totalBatches int
	if err := runtimeValues.Get(ctx, "preview_total_batches", &totalBatches); err != nil || totalBatches <= 0 {
		totalBatches = 1
	}
	return summary, totalBatches, true
}

func (s *previewService) selectItems(ctx context.Context, stateStore StateStore, input Input, req PreviewRequest, totalBatches int) (*int, []json.RawMessage, map[string]any, error) {
	if req.BatchIndex != nil {
		batch, err := stateStore.LoadBatch(ctx, fmt.Sprintf("%s:batches", input.RedisKey), *req.BatchIndex)
		if err != nil {
			return req.BatchIndex, nil, nil, err
		}
		return req.BatchIndex, limitItems(batch.Items, req.Offset, req.Limit), map[string]any{
			"offset":      req.Offset,
			"limit":       normalizeLimit(req.Limit),
			"total_items": len(batch.Items),
		}, nil
	}

	items := make([]json.RawMessage, 0)
	for batchIndex := 0; batchIndex < totalBatches; batchIndex++ {
		batch, err := stateStore.LoadBatch(ctx, fmt.Sprintf("%s:batches", input.RedisKey), batchIndex)
		if err != nil {
			return nil, nil, nil, err
		}
		items = append(items, batch.Items...)
	}
	selected := limitItems(items, req.Offset, req.Limit)
	return nil, selected, map[string]any{
		"offset":      req.Offset,
		"limit":       normalizeLimit(req.Limit),
		"total_items": len(items),
	}, nil
}

func limitItems(items []json.RawMessage, offset int, limit int) []json.RawMessage {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 20
	}
	if offset >= len(items) {
		return []json.RawMessage{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end]
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	return limit
}

func normalizePreviewMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "":
		return "all"
	case "prepare", "batch", "all":
		return strings.ToLower(strings.TrimSpace(mode))
	default:
		return "all"
	}
}

func composePreviewRedisKey(processTypeID int64, resolvedProcessVersionID int64, redisKey string) string {
	base := strings.TrimSpace(redisKey)
	if base == "" {
		base = uuid.NewString()
	}
	if resolvedProcessVersionID > 0 {
		return fmt.Sprintf("preview:%d:%d:%s", processTypeID, resolvedProcessVersionID, base)
	}
	return fmt.Sprintf("preview:%d:%s", processTypeID, base)
}

var (
	defaultPreviewSvcOnce sync.Once
	defaultPreviewSvc     PreviewService
)

func DefaultPreviewService() (PreviewService, error) {
	defaultPreviewSvcOnce.Do(func() {
		defaultPreviewSvc = NewPreviewService(defaultPreviewRegistry, 30*time.Minute)
	})
	return defaultPreviewSvc, nil
}
