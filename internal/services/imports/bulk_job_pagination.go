package imports

import (
	"context"
	"fmt"
	"strings"

	"go-fiber-core/internal/dtos"
	"go-fiber-core/internal/models"

	"gorm.io/gorm"
)

type bulkJobPaginationExtras struct {
	TotalRecords                int64
	PendingRecords              int64
	ProcessedRecords            int64
	ErrorRecords                int64
	ProcessedWithDetailsRecords int64
}

func (s *service) GetBulkJobsPaginated(ctx context.Context, req dtos.PaginationRequest) (*dtos.PaginationResponse[models.BulkJobListItem], error) {
	db := s.conn.ConnectGormRead
	if db == nil {
		return nil, fmt.Errorf("gorm read connection is not initialized")
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.RowsPerPage <= 0 {
		req.RowsPerPage = 15
	}

	query := s.buildBulkJobsPaginationQuery(ctx)
	query = applyBulkJobPaginationFilters(query, req)

	var totalRows int64
	if err := query.Count(&totalRows).Error; err != nil {
		return nil, err
	}

	extras, err := calculateBulkJobPaginationExtras(query.Session(&gorm.Session{}))
	if err != nil {
		return nil, err
	}

	if totalRows == 0 {
		return &dtos.PaginationResponse[models.BulkJobListItem]{
			Data:        []models.BulkJobListItem{},
			TotalRows:   0,
			TotalPages:  0,
			Page:        req.Page,
			RowsPerPage: req.RowsPerPage,
			Extras:      buildBulkJobExtrasMap(extras),
		}, nil
	}

	query = applyBulkJobPaginationSort(query, req)

	offset := (req.Page - 1) * req.RowsPerPage
	var items []models.BulkJobListItem
	if err := query.Limit(req.RowsPerPage).Offset(offset).Scan(&items).Error; err != nil {
		return nil, err
	}

	totalPages := int((totalRows + int64(req.RowsPerPage) - 1) / int64(req.RowsPerPage))
	return &dtos.PaginationResponse[models.BulkJobListItem]{
		Data:        items,
		TotalRows:   totalRows,
		TotalPages:  totalPages,
		Page:        req.Page,
		RowsPerPage: req.RowsPerPage,
		Extras:      buildBulkJobExtrasMap(extras),
	}, nil
}

func (s *service) buildBulkJobsPaginationQuery(ctx context.Context) *gorm.DB {
	metrics := s.conn.ConnectGormRead.WithContext(ctx).
		Table("bulk_job_items bji").
		Select(`
			bji.bulk_job_id AS bulk_job_id,
			COUNT(*) AS total_records,
			COUNT(*) FILTER (WHERE bji.status_code = 'IMPORTED') AS pending_records,
			COUNT(*) FILTER (WHERE bji.status_code = 'PROCESSED') AS processed_records,
			COUNT(*) FILTER (WHERE bji.status_code = 'ERROR_PROCESS') AS error_records,
			COUNT(*) FILTER (WHERE bji.status_code = 'PROCESSED_WITH_DETAILS') AS processed_with_details_records
		`).
		Group("bji.bulk_job_id")

	base := s.conn.ConnectGormRead.WithContext(ctx).
		Table("bulk_jobs bj").
		Select(`
			bj.id AS id,
			bj.operator_id AS operator_id,
			bj.branch_id AS branch_id,
			bj.key_code AS key_code,
			bj.ref_code AS ref_code,
			bj.status_code AS status_code,
			bj.total_detail_items AS declared_total_records,
			bj.file_name AS file_name,
			bj.created_at AS created_at,
			bj.updated_at AS updated_at,
			COALESCE(metrics.total_records, 0) AS total_records,
			COALESCE(metrics.pending_records, 0) AS pending_records,
			COALESCE(metrics.processed_records, 0) AS processed_records,
			COALESCE(metrics.error_records, 0) AS error_records,
			COALESCE(metrics.processed_with_details_records, 0) AS processed_with_details_records
		`).
		Joins("LEFT JOIN (?) metrics ON metrics.bulk_job_id = bj.id", metrics)

	return s.conn.ConnectGormRead.WithContext(ctx).
		Table("(?) AS bulk_jobs_paginated", base)
}

func applyBulkJobPaginationFilters(query *gorm.DB, req dtos.PaginationRequest) *gorm.DB {
	for i, filterBy := range req.FilterBy {
		if i >= len(req.FilterValues) {
			continue
		}
		value := req.FilterValues[i]
		switch filterBy {
		case "id":
			query = applyPaginationInt64Filter(query, "id", value)
		case "operator_id":
			query = applyPaginationUint64Filter(query, "operator_id", value)
		case "branch_id":
			query = applyPaginationInt64Filter(query, "branch_id", value)
		case "key_code":
			query = applyPaginationStringFilter(query, "key_code", value, true)
		case "ref_code":
			query = applyPaginationStringFilter(query, "ref_code", value, true)
		case "status_code":
			query = applyPaginationStringFilter(query, "status_code", value, false)
		case "file_name":
			query = applyPaginationStringFilter(query, "file_name", value, true)
		case "total_records":
			query = applyPaginationInt64Filter(query, "total_records", value)
		case "pending_records":
			query = applyPaginationInt64Filter(query, "pending_records", value)
		case "processed_records":
			query = applyPaginationInt64Filter(query, "processed_records", value)
		case "error_records":
			query = applyPaginationInt64Filter(query, "error_records", value)
		case "processed_with_details_records":
			query = applyPaginationInt64Filter(query, "processed_with_details_records", value)
		}
	}
	return query
}

func applyBulkJobPaginationSort(query *gorm.DB, req dtos.PaginationRequest) *gorm.DB {
	if len(req.SortBy) == 0 {
		return query.Order("id DESC")
	}
	for i, sortBy := range req.SortBy {
		column := ""
		switch sortBy {
		case "id", "operator_id", "branch_id", "key_code", "ref_code", "status_code", "file_name", "created_at", "updated_at":
			column = sortBy
		case "total_records", "pending_records", "processed_records", "error_records", "processed_with_details_records", "declared_total_records":
			column = sortBy
		default:
			continue
		}

		direction := "ASC"
		if i < len(req.SortDesc) && req.SortDesc[i] {
			direction = "DESC"
		}
		query = query.Order(column + " " + direction)
	}
	return query
}

func calculateBulkJobPaginationExtras(query *gorm.DB) (bulkJobPaginationExtras, error) {
	var extras bulkJobPaginationExtras
	err := query.Select(`
		COALESCE(SUM(total_records), 0) AS total_records,
		COALESCE(SUM(pending_records), 0) AS pending_records,
		COALESCE(SUM(processed_records), 0) AS processed_records,
		COALESCE(SUM(error_records), 0) AS error_records,
		COALESCE(SUM(processed_with_details_records), 0) AS processed_with_details_records
	`).Scan(&extras).Error
	return extras, err
}

func buildBulkJobExtrasMap(extras bulkJobPaginationExtras) map[string]any {
	return map[string]any{
		"total_records":                  extras.TotalRecords,
		"pending_records":                extras.PendingRecords,
		"processed_records":              extras.ProcessedRecords,
		"error_records":                  extras.ErrorRecords,
		"processed_with_details_records": extras.ProcessedWithDetailsRecords,
	}
}

func applyPaginationStringFilter(query *gorm.DB, column string, value any, fuzzy bool) *gorm.DB {
	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return query
		}
		if fuzzy {
			return query.Where(column+" ILIKE ?", "%"+strings.TrimSpace(typed)+"%")
		}
		return query.Where(column+" = ?", strings.TrimSpace(typed))
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			if str, ok := item.(string); ok && strings.TrimSpace(str) != "" {
				values = append(values, strings.TrimSpace(str))
			}
		}
		if len(values) > 0 {
			return query.Where(column+" IN ?", values)
		}
	}
	return query
}

func applyPaginationInt64Filter(query *gorm.DB, column string, value any) *gorm.DB {
	values := normalizeInt64Values(value)
	switch len(values) {
	case 0:
		return query
	case 1:
		return query.Where(column+" = ?", values[0])
	default:
		return query.Where(column+" IN ?", values)
	}
}

func applyPaginationUint64Filter(query *gorm.DB, column string, value any) *gorm.DB {
	values := normalizeUint64Values(value)
	switch len(values) {
	case 0:
		return query
	case 1:
		return query.Where(column+" = ?", values[0])
	default:
		return query.Where(column+" IN ?", values)
	}
}

func normalizeInt64Values(value any) []int64 {
	switch typed := value.(type) {
	case int:
		return []int64{int64(typed)}
	case int64:
		return []int64{typed}
	case float64:
		return []int64{int64(typed)}
	case []any:
		values := make([]int64, 0, len(typed))
		for _, item := range typed {
			values = append(values, normalizeInt64Values(item)...)
		}
		return values
	}
	return nil
}

func normalizeUint64Values(value any) []uint64 {
	switch typed := value.(type) {
	case uint:
		return []uint64{uint64(typed)}
	case uint64:
		return []uint64{typed}
	case int:
		if typed >= 0 {
			return []uint64{uint64(typed)}
		}
	case int64:
		if typed >= 0 {
			return []uint64{uint64(typed)}
		}
	case float64:
		if typed >= 0 {
			return []uint64{uint64(typed)}
		}
	case []any:
		values := make([]uint64, 0, len(typed))
		for _, item := range typed {
			values = append(values, normalizeUint64Values(item)...)
		}
		return values
	}
	return nil
}
