package bulkjobconfig

import (
	"context"

	"go-fiber-core/internal/models"

	"gorm.io/gorm"
)

type BulkJobConfigReader interface {
	GetActiveByRefCode(ctx context.Context, db *gorm.DB, operatorID uint64, refCode string) (*models.BulkJobConfig, error)
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
