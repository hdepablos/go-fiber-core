package bulkexportv2

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"go-fiber-core/internal/models"
	"go-fiber-core/internal/services/exportmanager"

	"gorm.io/gorm"
)

type BulkJobDataProvider struct {
	readDB *gorm.DB
}

func NewBulkJobDataProvider(readDB *gorm.DB) *BulkJobDataProvider {
	return &BulkJobDataProvider{readDB: readDB}
}

func (p *BulkJobDataProvider) LoadBatches(ctx context.Context, execCtx exportmanager.ExecutionContext, batchSize int) (exportmanager.LoadBatchesResult, error) {
	input := execCtx.Input
	if input.ParentID <= 0 {
		return exportmanager.LoadBatchesResult{}, fmt.Errorf("id inválido")
	}
	if batchSize <= 0 {
		batchSize = 5000
	}

	query := p.readDB.WithContext(ctx).
		Model(&models.BulkJobItem{}).
		Select("id", "row_number", "reference_key", "data", "created_at", "updated_at").
		Where("bulk_job_id = ?", input.ParentID).
		Order("id ASC")

	statusFilterApplied := false
	if input.Filters != nil {
		var err error
		query, statusFilterApplied, err = applyBulkJobItemFilters(query, input.Filters)
		if err != nil {
			return exportmanager.LoadBatchesResult{}, err
		}
	}
	if !statusFilterApplied {
		query = query.Where("status_code = ?", models.BulkJobStatusImported)
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
			"id":            item.ID,
			"row_number":    item.RowNumber,
			"reference_key": item.ReferenceKey,
			"data":          json.RawMessage(item.Data),
		})
		if err != nil {
			return exportmanager.LoadBatchesResult{}, err
		}
		current.Items = append(current.Items, payload)
		totalAmount += extractAmount(item.Data)

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

func extractAmount(raw []byte) float64 {
	if len(raw) == 0 {
		return 0
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return 0
	}
	val, ok := data["amount"]
	if !ok {
		return 0
	}
	switch n := val.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case string:
		f, _ := strconv.ParseFloat(n, 64)
		return f
	default:
		return 0
	}
}

func applyBulkJobItemFilters(query *gorm.DB, rawFilters any) (*gorm.DB, bool, error) {
	filters, err := exportmanager.NormalizeFilters(rawFilters)
	if err != nil {
		return nil, false, err
	}
	if len(filters) == 0 {
		return query, false, nil
	}

	statusFilterApplied := false

	for key, value := range filters {
		switch key {
		case "bulk_job_items.status_code", "status_code":
			query = applyStringFilter(query, "status_code", value)
			statusFilterApplied = true
		case "bulk_job_items.reference_key", "reference_key":
			query = applyStringFilter(query, "reference_key", value)
		case "bulk_job_items.row_number", "row_number":
			query = applyIntFilter(query, "row_number", value)
		case "bulk_job_items.id", "id":
			query = applyInt64Filter(query, "id", value)
		case "bulk_job_items.bulk_job_id", "bulk_job_id":
			query = applyInt64Filter(query, "bulk_job_id", value)
		}
	}

	return query, statusFilterApplied, nil
}

func applyStringFilter(query *gorm.DB, field string, value any) *gorm.DB {
	switch typed := value.(type) {
	case string:
		if typed != "" {
			return query.Where(field+" = ?", typed)
		}
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			if str, ok := item.(string); ok && str != "" {
				values = append(values, str)
			}
		}
		if len(values) > 0 {
			return query.Where(field+" IN ?", values)
		}
	}
	return query
}

func applyIntFilter(query *gorm.DB, field string, value any) *gorm.DB {
	switch typed := value.(type) {
	case int:
		return query.Where(field+" = ?", typed)
	case int64:
		return query.Where(field+" = ?", typed)
	case float64:
		return query.Where(field+" = ?", int(typed))
	case string:
		if n, err := strconv.Atoi(typed); err == nil {
			return query.Where(field+" = ?", n)
		}
	case []any:
		values := make([]int, 0, len(typed))
		for _, item := range typed {
			switch casted := item.(type) {
			case int:
				values = append(values, casted)
			case int64:
				values = append(values, int(casted))
			case float64:
				values = append(values, int(casted))
			case string:
				if n, err := strconv.Atoi(casted); err == nil {
					values = append(values, n)
				}
			}
		}
		if len(values) > 0 {
			return query.Where(field+" IN ?", values)
		}
	}
	return query
}

func applyInt64Filter(query *gorm.DB, field string, value any) *gorm.DB {
	switch typed := value.(type) {
	case int:
		return query.Where(field+" = ?", int64(typed))
	case int64:
		return query.Where(field+" = ?", typed)
	case float64:
		return query.Where(field+" = ?", int64(typed))
	case string:
		if n, err := strconv.ParseInt(typed, 10, 64); err == nil {
			return query.Where(field+" = ?", n)
		}
	case []any:
		values := make([]int64, 0, len(typed))
		for _, item := range typed {
			switch casted := item.(type) {
			case int:
				values = append(values, int64(casted))
			case int64:
				values = append(values, casted)
			case float64:
				values = append(values, int64(casted))
			case string:
				if n, err := strconv.ParseInt(casted, 10, 64); err == nil {
					values = append(values, n)
				}
			}
		}
		if len(values) > 0 {
			return query.Where(field+" IN ?", values)
		}
	}
	return query
}
