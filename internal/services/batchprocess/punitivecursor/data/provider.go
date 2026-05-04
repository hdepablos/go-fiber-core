package data

import (
	"context"
	"encoding/json"
	"fmt"

	"go-fiber-core/internal/models"
	"go-fiber-core/internal/services/batchflow"
	"go-fiber-core/internal/utils"

	"gorm.io/gorm"
)

type dataProvider struct {
	readDB *gorm.DB
}

func NewDataProvider(readDB *gorm.DB) batchflow.DataProvider {
	return &dataProvider{readDB: readDB}
}

// LoadBatches usa bulk_job_items como fuente de datos y los corta en lotes por bulk_job_id.
func (p *dataProvider) LoadBatches(ctx context.Context, execCtx batchflow.ExecutionContext, batchSize int) (batchflow.LoadBatchesResult, error) {
	input := execCtx.Input
	if input.ParentID <= 0 {
		return batchflow.LoadBatchesResult{}, fmt.Errorf("id invalido")
	}
	if batchSize <= 0 {
		batchSize = 500
	}

	query, err := p.baseQuery(ctx, input)
	if err != nil {
		return batchflow.LoadBatchesResult{}, err
	}

	// Se carga el universo completo y luego se parte en batches en memoria.
	var items []models.BulkJobItem
	if err := query.Find(&items).Error; err != nil {
		return batchflow.LoadBatchesResult{}, err
	}

	batches := make([]batchflow.Batch, 0, (len(items)/batchSize)+1)
	current := batchflow.Batch{Items: make([]json.RawMessage, 0, batchSize)}
	for _, item := range items {
		payload, err := marshalBulkJobItem(item)
		if err != nil {
			return batchflow.LoadBatchesResult{}, err
		}
		current.Items = append(current.Items, payload)
		if len(current.Items) == batchSize {
			batches = append(batches, current)
			current = batchflow.Batch{Items: make([]json.RawMessage, 0, batchSize)}
		}
	}
	if len(current.Items) > 0 {
		batches = append(batches, current)
	}

	// El Summary conserva el contexto agregado que luego usaran preview, finalize y cierre operativo.
	return batchflow.LoadBatchesResult{
		Batches: batches,
		Summary: batchflow.Summary{
			TotalRecords: int64(len(items)),
			Metadata: map[string]any{
				"source":      "bulk_job_items",
				"bulk_job_id": input.ParentID,
			},
		},
	}, nil
}

// PrepareCursorRun deja listo el modo incremental guardando el summary y el cursor inicial.
func (p *dataProvider) PrepareCursorRun(ctx context.Context, execCtx batchflow.ExecutionContext, batchSize int) (batchflow.CursorRunResult, error) {
	input := execCtx.Input
	if input.ParentID <= 0 {
		return batchflow.CursorRunResult{}, fmt.Errorf("id invalido")
	}
	if batchSize <= 0 {
		batchSize = 500
	}

	query, err := p.baseQuery(ctx, input)
	if err != nil {
		return batchflow.CursorRunResult{}, err
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return batchflow.CursorRunResult{}, err
	}

	return batchflow.CursorRunResult{
		Summary: batchflow.Summary{
			TotalRecords: total,
			Metadata: map[string]any{
				"source":       "bulk_job_items",
				"bulk_job_id":  input.ParentID,
				"source_mode":  batchflow.SourceModeCursor,
				"batch_size":   batchSize,
				"initial_last": 0,
			},
		},
		Metadata: map[string]any{
			"source_mode": batchflow.SourceModeCursor,
		},
		InitialCursor: map[string]any{
			"last_id": int64(0),
		},
	}, nil
}

// LoadCursorBatch carga la siguiente pagina por id ascendente sin materializar todo el universo.
func (p *dataProvider) LoadCursorBatch(ctx context.Context, execCtx batchflow.ExecutionContext, req batchflow.CursorBatchRequest) (batchflow.CursorBatchResult, error) {
	input := execCtx.Input
	if input.ParentID <= 0 {
		return batchflow.CursorBatchResult{}, fmt.Errorf("id invalido")
	}
	if req.BatchSize <= 0 {
		req.BatchSize = 500
	}

	lastID := readCursorLastID(req.Cursor)
	query, err := p.baseQuery(ctx, input)
	if err != nil {
		return batchflow.CursorBatchResult{}, err
	}
	query = query.Where("id > ?", lastID).Limit(req.BatchSize + 1)

	var items []models.BulkJobItem
	if err := query.Find(&items).Error; err != nil {
		return batchflow.CursorBatchResult{}, err
	}
	if len(items) == 0 {
		return batchflow.CursorBatchResult{
			Batch: batchflow.Batch{Items: []json.RawMessage{}},
			NextCursor: map[string]any{
				"last_id": lastID,
			},
			HasMore: false,
			Metadata: map[string]any{
				"loaded_count": 0,
				"last_id":      lastID,
			},
		}, nil
	}

	hasMore := len(items) > req.BatchSize
	if hasMore {
		items = items[:req.BatchSize]
	}

	batchItems := make([]json.RawMessage, 0, len(items))
	for _, item := range items {
		payload, err := marshalBulkJobItem(item)
		if err != nil {
			return batchflow.CursorBatchResult{}, err
		}
		batchItems = append(batchItems, payload)
	}

	nextLastID := items[len(items)-1].ID
	return batchflow.CursorBatchResult{
		Batch: batchflow.Batch{Items: batchItems},
		NextCursor: map[string]any{
			"last_id": nextLastID,
		},
		HasMore: hasMore,
		Metadata: map[string]any{
			"loaded_count": len(batchItems),
			"last_id":      lastID,
			"next_last_id": nextLastID,
		},
	}, nil
}

func (p *dataProvider) baseQuery(ctx context.Context, input batchflow.Input) (*gorm.DB, error) {
	query := p.readDB.WithContext(ctx).
		Model(&models.BulkJobItem{}).
		Select("id", "bulk_job_id", "row_number", "reference_key", "status_code", "last_detail_message", "data", "created_at", "updated_at").
		Where("bulk_job_id = ?", input.ParentID).
		Order("id ASC")

	statusFilterApplied := false
	if input.Filters != nil {
		result, err := utils.ApplyBulkJobItemFilters(query, input.Filters)
		if err != nil {
			return nil, err
		}
		query = result.Query
		statusFilterApplied = result.StatusFilterApplied
	}
	if !statusFilterApplied {
		query = query.Where("status_code = ?", models.BulkJobStatusImported)
	}
	return query, nil
}

func marshalBulkJobItem(item models.BulkJobItem) (json.RawMessage, error) {
	payload, err := json.Marshal(map[string]any{
		"id":                  item.ID,
		"bulk_job_id":         item.BulkJobID,
		"row_number":          item.RowNumber,
		"reference_key":       item.ReferenceKey,
		"status_code":         item.StatusCode,
		"last_detail_message": item.LastDetailMessage,
		"data":                json.RawMessage(item.Data),
	})
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func readCursorLastID(cursor map[string]any) int64 {
	if len(cursor) == 0 {
		return 0
	}
	switch value := cursor["last_id"].(type) {
	case int64:
		return value
	case int32:
		return int64(value)
	case int:
		return int64(value)
	case float64:
		return int64(value)
	case float32:
		return int64(value)
	default:
		return 0
	}
}
