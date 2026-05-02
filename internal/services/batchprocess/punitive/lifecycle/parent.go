package lifecycle

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

// Start cambia el status del padre a PROCESSING antes de comenzar a procesar lotes.
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

// End persiste el status final del bulk job usando lo calculado en Finalize.
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

// Fail marca error de proceso en el padre cuando un step falla de forma no recuperable.
func (l *parentLifecycle) Fail(ctx context.Context, execCtx batchflow.ExecutionContext, _ error) error {
	return l.writeDB.WithContext(ctx).
		Model(&models.BulkJob{}).
		Where("id = ?", execCtx.Input.ParentID).
		Update("status_code", models.BulkJobStatusErrorProcess).Error
}

// RefreshProgress queda disponible para persistir avance incremental del padre.
func (l *parentLifecycle) RefreshProgress(ctx context.Context, execCtx batchflow.ExecutionContext, batch batchflow.Batch) error {
	_ = ctx
	_ = execCtx
	_ = batch
	return nil
}
