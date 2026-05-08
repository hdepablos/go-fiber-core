package lifecycle

import (
	"context"
	"fmt"

	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/models"
	"go-fiber-core/internal/services/exportmanager"

	"gorm.io/gorm"
)

type parentLifecycle struct {
	readDB  *gorm.DB
	writeDB *gorm.DB
}

// NewParentLifecycle concentra los cambios de status del padre durante todo el export.
func NewParentLifecycle(readDB, writeDB *gorm.DB) exportmanager.ParentLifecycle {
	return &parentLifecycle{readDB: readDB, writeDB: writeDB}
}

// Start valida el padre y lo mueve a PROCESSING antes de iniciar el export.
func (l *parentLifecycle) Start(ctx context.Context, execCtx exportmanager.ExecutionContext) error {
	input := execCtx.Input
	var job models.BulkJob
	if err := l.readDB.WithContext(ctx).
		Select("id", "status_code").
		Where("id = ?", input.ParentID).
		First(&job).Error; err != nil {
		return err
	}
	if job.StatusCode != models.BulkJobStatusImported {
		return fmt.Errorf("%w: Verifique el proceso con el id %d ya fue procesado actualmente con el status %s de la tabla bulk_jobs", domain.ErrBusinessRuleViolation, input.ParentID, job.StatusCode)
	}
	return l.writeDB.WithContext(ctx).
		Model(&models.BulkJob{}).
		Where("id = ?", input.ParentID).
		Update("status_code", models.BulkJobStatusProcessing).Error
}

// End persiste el status final exitoso del padre cuando el archivo ya fue generado.
func (l *parentLifecycle) End(ctx context.Context, execCtx exportmanager.ExecutionContext, _ exportmanager.OutputResult) error {
	return l.writeDB.WithContext(ctx).
		Model(&models.BulkJob{}).
		Where("id = ?", execCtx.Input.ParentID).
		Update("status_code", models.BulkJobStatusProcessed).Error
}

// Fail marca error de proceso en el padre cuando el export no puede completarse.
func (l *parentLifecycle) Fail(ctx context.Context, execCtx exportmanager.ExecutionContext, _ error) error {
	return l.writeDB.WithContext(ctx).
		Model(&models.BulkJob{}).
		Where("id = ?", execCtx.Input.ParentID).
		Update("status_code", models.BulkJobStatusErrorProcess).Error
}
