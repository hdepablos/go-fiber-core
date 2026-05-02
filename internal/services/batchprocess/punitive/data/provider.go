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

	// La consulta base define la fuente de datos del proceso.
	query := p.readDB.WithContext(ctx).
		Model(&models.BulkJobItem{}).
		Select("id", "bulk_job_id", "row_number", "reference_key", "status_code", "last_detail_message", "data", "created_at", "updated_at").
		Where("bulk_job_id = ?", input.ParentID).
		Order("id ASC")

	statusFilterApplied := false
	if input.Filters != nil {
		result, err := utils.ApplyBulkJobItemFilters(query, input.Filters)
		if err != nil {
			return batchflow.LoadBatchesResult{}, err
		}
		query = result.Query
		statusFilterApplied = result.StatusFilterApplied
	}
	if !statusFilterApplied {
		query = query.Where("status_code = ?", models.BulkJobStatusImported)
	}

	// Se carga el universo completo y luego se parte en batches en memoria.
	var items []models.BulkJobItem
	if err := query.Find(&items).Error; err != nil {
		return batchflow.LoadBatchesResult{}, err
	}

	batches := make([]batchflow.Batch, 0, (len(items)/batchSize)+1)
	current := batchflow.Batch{Items: make([]json.RawMessage, 0, batchSize)}
	for _, item := range items {
		// Cada item del detalle se serializa para que el processor lo reciba desacoplado de GORM.
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
