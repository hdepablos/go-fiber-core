package bulkjob

import (
	"context"

	"go-fiber-core/internal/models"

	"gorm.io/gorm"
)

type BulkJobReader interface {
	GetByID(ctx context.Context, db *gorm.DB, id int64) (*models.BulkJob, error)
	GetByKeyCode(ctx context.Context, db *gorm.DB, keyCode string) (*models.BulkJob, error)
}

type BulkJobWriter interface {
	Create(ctx context.Context, db *gorm.DB, job *models.BulkJob) error
	UpdateStatus(ctx context.Context, db *gorm.DB, id int64, status models.BulkJobStatus) error
	IncrementTotalDetailItems(ctx context.Context, db *gorm.DB, id int64, delta int) error
	IncrementTotalProcessedItems(ctx context.Context, db *gorm.DB, id int64, delta int) error
}

type bulkJobReaderRepo struct{}

func NewBulkJobReaderRepo() BulkJobReader { return &bulkJobReaderRepo{} }

func (r *bulkJobReaderRepo) GetByID(ctx context.Context, db *gorm.DB, id int64) (*models.BulkJob, error) {
	var job models.BulkJob
	if err := db.WithContext(ctx).First(&job, id).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *bulkJobReaderRepo) GetByKeyCode(ctx context.Context, db *gorm.DB, keyCode string) (*models.BulkJob, error) {
	var job models.BulkJob
	if err := db.WithContext(ctx).Where("key_code = ?", keyCode).First(&job).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

type bulkJobWriterRepo struct{}

func NewBulkJobWriterRepo() BulkJobWriter { return &bulkJobWriterRepo{} }

func (w *bulkJobWriterRepo) Create(ctx context.Context, db *gorm.DB, job *models.BulkJob) error {
	return db.WithContext(ctx).Create(job).Error
}

func (w *bulkJobWriterRepo) UpdateStatus(ctx context.Context, db *gorm.DB, id int64, status models.BulkJobStatus) error {
	return db.WithContext(ctx).Model(&models.BulkJob{}).Where("id = ?", id).Update("status_code", status).Error
}

func (w *bulkJobWriterRepo) IncrementTotalDetailItems(ctx context.Context, db *gorm.DB, id int64, delta int) error {
	return db.WithContext(ctx).Model(&models.BulkJob{}).Where("id = ?", id).UpdateColumn("total_detail_items", gorm.Expr("total_detail_items + ?", delta)).Error
}

func (w *bulkJobWriterRepo) IncrementTotalProcessedItems(ctx context.Context, db *gorm.DB, id int64, delta int) error {
	return db.WithContext(ctx).Model(&models.BulkJob{}).Where("id = ?", id).UpdateColumn("total_processed_items", gorm.Expr("total_processed_items + ?", delta)).Error
}
