package bulkjobitem

import (
	"context"
	"errors"

	"go-fiber-core/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"gorm.io/gorm"
)

type BulkJobItemReader interface {
	GetByID(ctx context.Context, db *gorm.DB, id int64) (*models.BulkJobItem, error)
	GetMaxRowNumber(ctx context.Context, db *gorm.DB, bulkJobID int64) (int, error)
}

type BulkJobItemWriter interface {
	BulkCreate(ctx context.Context, db *gorm.DB, items []*models.BulkJobItem) error
	BulkCreatePGX(ctx context.Context, pool *pgxpool.Pool, items []*models.BulkJobItem) error
	UpdateStatus(ctx context.Context, db *gorm.DB, id int64, status models.BulkJobStatus, lastDetailMessage *string) error
}

type bulkJobItemReaderRepo struct{}

func NewBulkJobItemReaderRepo() BulkJobItemReader { return &bulkJobItemReaderRepo{} }

func (r *bulkJobItemReaderRepo) GetByID(ctx context.Context, db *gorm.DB, id int64) (*models.BulkJobItem, error) {
	var item models.BulkJobItem
	if err := db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *bulkJobItemReaderRepo) GetMaxRowNumber(ctx context.Context, db *gorm.DB, bulkJobID int64) (int, error) {
	var max int
	if err := db.WithContext(ctx).
		Model(&models.BulkJobItem{}).
		Where("bulk_job_id = ?", bulkJobID).
		Select("COALESCE(MAX(row_number), 0)").
		Scan(&max).Error; err != nil {
		return 0, err
	}
	return max, nil
}

type bulkJobItemWriterRepo struct{}

func NewBulkJobItemWriterRepo() BulkJobItemWriter { return &bulkJobItemWriterRepo{} }

func (w *bulkJobItemWriterRepo) BulkCreate(ctx context.Context, db *gorm.DB, items []*models.BulkJobItem) error {
	return db.WithContext(ctx).Create(&items).Error
}

func (w *bulkJobItemWriterRepo) BulkCreatePGX(ctx context.Context, pool *pgxpool.Pool, items []*models.BulkJobItem) error {
	if pool == nil {
		return errors.New("pgx pool is nil")
	}
	if len(items) == 0 {
		return nil
	}

	rows := make([][]any, 0, len(items))
	for _, it := range items {
		if it == nil {
			continue
		}
		rows = append(rows, []any{
			it.BulkJobID,
			it.RowNumber,
			it.ReferenceKey,
			it.Data,
			string(it.StatusCode),
			it.LastDetailMessage,
		})
	}

	_, err := pool.CopyFrom(
		ctx,
		pgx.Identifier{"bulk_job_items"},
		[]string{"bulk_job_id", "row_number", "reference_key", "data", "status_code", "last_detail_message"},
		pgx.CopyFromRows(rows),
	)
	return err
}

func (w *bulkJobItemWriterRepo) UpdateStatus(ctx context.Context, db *gorm.DB, id int64, status models.BulkJobStatus, lastDetailMessage *string) error {
	updates := map[string]interface{}{
		"status_code": status,
	}
	if lastDetailMessage != nil {
		updates["last_detail_message"] = *lastDetailMessage
	}
	return db.WithContext(ctx).Model(&models.BulkJobItem{}).Where("id = ?", id).Updates(updates).Error
}
