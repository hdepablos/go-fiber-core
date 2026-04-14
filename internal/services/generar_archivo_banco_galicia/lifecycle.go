package generar_archivo_banco_galicia

import (
	"context"
	"encoding/json"
	"fmt"

	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/models"
	"go-fiber-core/internal/services/exportmanager"

	"gorm.io/gorm"
)

type ParentLifecycle struct {
	readDB  *gorm.DB
	writeDB *gorm.DB
}

func NewParentLifecycle(readDB, writeDB *gorm.DB) *ParentLifecycle {
	return &ParentLifecycle{readDB: readDB, writeDB: writeDB}
}

func (l *ParentLifecycle) Start(ctx context.Context, execCtx exportmanager.ExecutionContext) error {
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

func (l *ParentLifecycle) End(ctx context.Context, execCtx exportmanager.ExecutionContext, _ exportmanager.OutputResult) error {
	return l.writeDB.WithContext(ctx).
		Model(&models.BulkJob{}).
		Where("id = ?", execCtx.Input.ParentID).
		Update("status_code", models.BulkJobStatusProcessed).Error
}

func (l *ParentLifecycle) Fail(ctx context.Context, execCtx exportmanager.ExecutionContext, _ error) error {
	return l.writeDB.WithContext(ctx).
		Model(&models.BulkJob{}).
		Where("id = ?", execCtx.Input.ParentID).
		Update("status_code", models.BulkJobStatusErrorProcess).Error
}

type OutputRegistrar struct {
	writeDB *gorm.DB
}

func NewOutputRegistrar(writeDB *gorm.DB) *OutputRegistrar {
	return &OutputRegistrar{writeDB: writeDB}
}

func (r *OutputRegistrar) Register(ctx context.Context, execCtx exportmanager.ExecutionContext, output exportmanager.OutputResult) error {
	metadata, err := json.Marshal(map[string]any{
		"bucket":        output.Bucket,
		"key":           output.Key,
		"content_type":  output.ContentType,
		"parts":         output.Parts,
		"total_records": execCtx.Summary.TotalRecords,
		"total_amount":  execCtx.Summary.TotalAmount,
		"redis_key":     execCtx.Input.RedisKey,
	})
	if err != nil {
		return err
	}

	fileSize := output.FileSize
	record := &models.BulkJobOutput{
		BulkJobID: execCtx.Input.ParentID,
		Type:      "csv",
		FilePath:  output.Path,
		FileSize:  &fileSize,
		Status:    models.BulkJobOutputStatusGenerated,
		Metadata:  metadata,
	}
	return r.writeDB.WithContext(ctx).Create(record).Error
}
