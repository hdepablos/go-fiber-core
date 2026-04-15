package generar_archivo_banco_galicia

import (
	"context"
	"encoding/json"
	"fmt"

	"go-fiber-core/internal/models"
	"go-fiber-core/internal/services/exportmanager"
	"go-fiber-core/internal/utils"

	"gorm.io/gorm"
)

type dataProvider struct {
	readDB *gorm.DB
}

func NewDataProvider(readDB *gorm.DB) exportmanager.DataProvider {
	return &dataProvider{readDB: readDB}
}

func (p *dataProvider) LoadBatches(ctx context.Context, execCtx exportmanager.ExecutionContext, batchSize int) (exportmanager.LoadBatchesResult, error) {
	input := execCtx.Input
	if input.ParentID <= 0 {
		return exportmanager.LoadBatchesResult{}, fmt.Errorf("id invalido")
	}
	if batchSize <= 0 {
		batchSize = 5000
	}

	query := p.readDB.WithContext(ctx).
		Model(&models.BulkJobItem{}).
		Select("id", "bulk_job_id", "row_number", "reference_key", "status_code", "last_detail_message", "data", "created_at", "updated_at").
		Where("bulk_job_id = ?", input.ParentID).
		Order("id ASC")

	if input.Filters != nil {
		result, err := utils.ApplyBulkJobItemFilters(query, input.Filters)
		if err != nil {
			return exportmanager.LoadBatchesResult{}, err
		}
		query = result.Query
	}

	var items []models.BulkJobItem
	if err := query.Find(&items).Error; err != nil {
		return exportmanager.LoadBatchesResult{}, err
	}

	batches := make([]exportmanager.Batch, 0, (len(items)/batchSize)+1)
	var current exportmanager.Batch
	current.Items = make([]json.RawMessage, 0, batchSize)

	totalAmount := 0.0
	for _, item := range items {
		payload, err := json.Marshal(map[string]any{
			"ID":                item.ID,
			"BulkJobID":         item.BulkJobID,
			"RowNumber":         item.RowNumber,
			"ReferenceKey":      item.ReferenceKey,
			"StatusCode":        item.StatusCode,
			"LastDetailMessage": item.LastDetailMessage,
			"CreatedAt":         item.CreatedAt,
			"UpdatedAt":         item.UpdatedAt,
			"Data":              json.RawMessage(item.Data),
		})
		if err != nil {
			return exportmanager.LoadBatchesResult{}, err
		}
		current.Items = append(current.Items, payload)
		totalAmount += utils.ExtractAmount(item.Data)

		if len(current.Items) == batchSize {
			batches = append(batches, current)
			current = exportmanager.Batch{Items: make([]json.RawMessage, 0, batchSize)}
		}
	}
	if len(current.Items) > 0 {
		batches = append(batches, current)
	}

	if execCtx.Runtime != nil {
		_ = execCtx.Runtime.Set(ctx, "total_records", len(items))
		_ = execCtx.Runtime.Set(ctx, "total_amount", totalAmount)
	}

	return exportmanager.LoadBatchesResult{
		Batches: batches,
		Summary: exportmanager.Summary{
			TotalRecords: int64(len(items)),
			TotalAmount:  totalAmount,
			Defaults: map[string]any{
				"total_records": len(items),
				"total_amount":  totalAmount,
			},
			Metadata: map[string]any{
				"source":      "bulk_job_items",
				"bulk_job_id": input.ParentID,
			},
		},
	}, nil
}
