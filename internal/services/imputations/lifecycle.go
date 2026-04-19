package imputations

import (
	"context"
	"fmt"

	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/models"
	"go-fiber-core/internal/services/batchflow"

	"gorm.io/gorm"
)

type parentLifecycle struct {
	readDB  *gorm.DB
	writeDB *gorm.DB
}

func NewParentLifecycle(readDB, writeDB *gorm.DB) batchflow.ParentLifecycle {
	return &parentLifecycle{readDB: readDB, writeDB: writeDB}
}

func (l *parentLifecycle) Start(ctx context.Context, execCtx batchflow.ExecutionContext) error {
	var job models.BulkJob
	if err := l.readDB.WithContext(ctx).
		Select("id", "status_code").
		Where("id = ?", execCtx.Input.ParentID).
		First(&job).Error; err != nil {
		return err
	}
	if job.StatusCode != models.BulkJobStatusImported {
		return fmt.Errorf("%w: el bulk_job %d tiene status %s", domain.ErrBusinessRuleViolation, execCtx.Input.ParentID, job.StatusCode)
	}
	return l.writeDB.WithContext(ctx).
		Model(&models.BulkJob{}).
		Where("id = ?", execCtx.Input.ParentID).
		Update("status_code", models.BulkJobStatusProcessing).Error
}

func (l *parentLifecycle) End(ctx context.Context, execCtx batchflow.ExecutionContext, result batchflow.FinalizeResult) error {
	status := models.BulkJobStatusProcessed
	if raw, ok := result.Metadata["bulk_job_status"].(string); ok && raw != "" {
		status = models.BulkJobStatus(raw)
	}
	updates := map[string]any{
		"status_code": status,
	}
	return l.writeDB.WithContext(ctx).
		Model(&models.BulkJob{}).
		Where("id = ?", execCtx.Input.ParentID).
		Updates(updates).Error
}

func (l *parentLifecycle) Fail(ctx context.Context, execCtx batchflow.ExecutionContext, _ error) error {
	return l.writeDB.WithContext(ctx).
		Model(&models.BulkJob{}).
		Where("id = ?", execCtx.Input.ParentID).
		Update("status_code", models.BulkJobStatusErrorProcess).Error
}

func (l *parentLifecycle) RefreshProgress(ctx context.Context, execCtx batchflow.ExecutionContext, batch batchflow.Batch) error {
	_ = ctx
	_ = execCtx
	_ = batch
	return nil
}

type finalizer struct {
	readDB *gorm.DB
}

func NewFinalizer(readDB *gorm.DB) batchflow.Finalizer {
	return &finalizer{readDB: readDB}
}

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
		if isProcessedBulkJobStatus(row.StatusCode) {
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

func bulkJobProcessedStatuses() []models.BulkJobStatus {
	return []models.BulkJobStatus{
		models.BulkJobStatusProcessed,
		models.BulkJobStatusProcessedWithDetails,
		models.BulkJobStatusErrorProcess,
	}
}

func isProcessedBulkJobStatus(status models.BulkJobStatus) bool {
	for _, candidate := range bulkJobProcessedStatuses() {
		if status == candidate {
			return true
		}
	}
	return false
}
