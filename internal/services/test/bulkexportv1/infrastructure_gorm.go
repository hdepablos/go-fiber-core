package bulkexportv1

import (
	"context"

	"go-fiber-core/internal/models"

	"gorm.io/gorm"
)

type GormBulkJobRepository struct {
	readDB  *gorm.DB
	writeDB *gorm.DB
}

func NewGormBulkJobRepository(readDB, writeDB *gorm.DB) *GormBulkJobRepository {
	return &GormBulkJobRepository{readDB: readDB, writeDB: writeDB}
}

func (r *GormBulkJobRepository) GetStatus(ctx context.Context, bulkJobID int64) (models.BulkJobStatus, error) {
	var job models.BulkJob
	if err := r.readDB.WithContext(ctx).
		Model(&models.BulkJob{}).
		Select("status_code").
		Where("id = ?", bulkJobID).
		First(&job).Error; err != nil {
		return "", err
	}
	return job.StatusCode, nil
}

func (r *GormBulkJobRepository) UpdateStatus(ctx context.Context, bulkJobID int64, status models.BulkJobStatus) error {
	return r.writeDB.WithContext(ctx).
		Model(&models.BulkJob{}).
		Where("id = ?", bulkJobID).
		Update("status_code", status).Error
}

type GormBulkJobItemRepository struct {
	db *gorm.DB
}

func NewGormBulkJobItemRepository(db *gorm.DB) *GormBulkJobItemRepository {
	return &GormBulkJobItemRepository{db: db}
}

func (r *GormBulkJobItemRepository) ListIDsAfter(ctx context.Context, bulkJobID int64, lastID int64, limit int) ([]int64, error) {
	var ids []int64
	err := r.db.WithContext(ctx).
		Model(&models.BulkJobItem{}).
		Where("bulk_job_id = ? AND status_code = ? AND id > ?", bulkJobID, models.BulkJobStatusImported, lastID).
		Order("id ASC").
		Limit(limit).
		Pluck("id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *GormBulkJobItemRepository) FindByIDs(ctx context.Context, bulkJobID int64, ids []int64) ([]models.BulkJobItem, error) {
	if len(ids) == 0 {
		return []models.BulkJobItem{}, nil
	}
	var items []models.BulkJobItem
	if err := r.db.WithContext(ctx).
		Model(&models.BulkJobItem{}).
		Select("id", "row_number", "reference_key", "data").
		Where("bulk_job_id = ? AND id IN ?", bulkJobID, ids).
		Order("id ASC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

type GormBulkJobOutputRepository struct {
	db *gorm.DB
}

func NewGormBulkJobOutputRepository(db *gorm.DB) *GormBulkJobOutputRepository {
	return &GormBulkJobOutputRepository{db: db}
}

func (r *GormBulkJobOutputRepository) Create(ctx context.Context, output *models.BulkJobOutput) error {
	return r.db.WithContext(ctx).Create(output).Error
}
