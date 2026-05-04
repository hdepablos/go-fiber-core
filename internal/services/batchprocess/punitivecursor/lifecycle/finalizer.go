package lifecycle

import (
	"context"

	"go-fiber-core/internal/models"
	"go-fiber-core/internal/services/batchflow"

	"gorm.io/gorm"
)

type finalizer struct {
	readDB *gorm.DB
}

func NewFinalizer(readDB *gorm.DB) batchflow.Finalizer {
	return &finalizer{readDB: readDB}
}

// Finalize relee el detalle para calcular counters y sugerir el status final del padre.
func (f *finalizer) Finalize(ctx context.Context, execCtx batchflow.ExecutionContext, req batchflow.FinalizeRequest) (batchflow.FinalizeResult, error) {
	_ = req

	var rows []struct {
		StatusCode models.BulkJobStatus
		Total      int64
	}
	if err := f.readDB.WithContext(ctx).
		Model(&models.BulkJobItem{}).
		Select("status_code, COUNT(*) as total").
		Where("bulk_job_id = ?", execCtx.Input.ParentID).
		Group("status_code").
		Scan(&rows).Error; err != nil {
		return batchflow.FinalizeResult{}, err
	}

	counters := map[models.BulkJobStatus]int64{}
	var totalProcessed int64
	for _, row := range rows {
		counters[row.StatusCode] = row.Total
		if row.StatusCode == models.BulkJobStatusProcessed ||
			row.StatusCode == models.BulkJobStatusProcessedWithDetails ||
			row.StatusCode == models.BulkJobStatusErrorProcess {
			totalProcessed += row.Total
		}
	}

	finalStatus := models.BulkJobStatusProcessed
	errorCount := counters[models.BulkJobStatusErrorProcess]
	detailCount := counters[models.BulkJobStatusProcessedWithDetails]
	processedCount := counters[models.BulkJobStatusProcessed]

	switch {
	case errorCount > 0 && processedCount == 0 && detailCount == 0:
		finalStatus = models.BulkJobStatusErrorProcess
	case errorCount > 0 || detailCount > 0:
		finalStatus = models.BulkJobStatusProcessedWithDetails
	}

	summary := execCtx.Summary
	summary.Metadata = map[string]any{
		"status_counters": map[string]int64{
			string(models.BulkJobStatusProcessed):            processedCount,
			string(models.BulkJobStatusErrorProcess):         errorCount,
			string(models.BulkJobStatusProcessedWithDetails): detailCount,
		},
	}

	return batchflow.FinalizeResult{
		Summary: summary,
		Metadata: map[string]any{
			"bulk_job_status": string(finalStatus),
			"processed_count": processedCount,
			"error_count":     errorCount,
			"detail_count":    detailCount,
			"pending_count":   counters[models.BulkJobStatusImported],
			"total_count":     totalProcessed + counters[models.BulkJobStatusImported],
		},
	}, nil
}
