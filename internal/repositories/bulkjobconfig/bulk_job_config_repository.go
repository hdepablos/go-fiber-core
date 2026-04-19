package bulkjobconfig

import (
	"context"
	"strconv"

	"go-fiber-core/internal/models"

	"gorm.io/gorm"
)

type BulkJobConfigReader interface {
	GetActiveByRefCode(ctx context.Context, db *gorm.DB, operatorID uint64, refCode string) (*models.BulkJobConfig, error)
	GetNextRefCode(ctx context.Context, db *gorm.DB, step int64) (string, error)
}

type BulkJobConfigWriter interface {
	Create(ctx context.Context, db *gorm.DB, cfg *models.BulkJobConfig) error
	DeactivateAllByRefCode(ctx context.Context, db *gorm.DB, operatorID uint64, refCode string) error
}

type bulkJobConfigReaderRepo struct{}

func NewBulkJobConfigReaderRepo() BulkJobConfigReader { return &bulkJobConfigReaderRepo{} }

func (r *bulkJobConfigReaderRepo) GetActiveByRefCode(ctx context.Context, db *gorm.DB, operatorID uint64, refCode string) (*models.BulkJobConfig, error) {
	var cfg models.BulkJobConfig
	if err := db.WithContext(ctx).
		Where("operator_id = ? AND ref_code = ? AND is_active = TRUE AND archived_at IS NULL", operatorID, refCode).
		First(&cfg).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (r *bulkJobConfigReaderRepo) GetNextRefCode(ctx context.Context, db *gorm.DB, step int64) (string, error) {
	if step <= 0 {
		step = 5
	}

	var maxValue int64
	if err := db.WithContext(ctx).
		Raw(`SELECT COALESCE(MAX(CAST(ref_code AS BIGINT)), 0) FROM bulk_job_configs WHERE ref_code ~ '^[0-9]+$'`).
		Scan(&maxValue).Error; err != nil {
		return "", err
	}

	return strconv.FormatInt(nextNumericRefCode(maxValue, step), 10), nil
}

func nextNumericRefCode(maxValue int64, step int64) int64 {
	if step <= 0 {
		step = 5
	}
	if maxValue <= 0 {
		return step
	}
	remainder := maxValue % step
	if remainder == 0 {
		return maxValue + step
	}
	return maxValue + (step - remainder)
}

type bulkJobConfigWriterRepo struct{}

func NewBulkJobConfigWriterRepo() BulkJobConfigWriter { return &bulkJobConfigWriterRepo{} }

func (w *bulkJobConfigWriterRepo) Create(ctx context.Context, db *gorm.DB, cfg *models.BulkJobConfig) error {
	return db.WithContext(ctx).Create(cfg).Error
}

func (w *bulkJobConfigWriterRepo) DeactivateAllByRefCode(ctx context.Context, db *gorm.DB, operatorID uint64, refCode string) error {
	return db.WithContext(ctx).
		Model(&models.BulkJobConfig{}).
		Where("operator_id = ? AND ref_code = ? AND archived_at IS NULL", operatorID, refCode).
		Update("is_active", false).Error
}
