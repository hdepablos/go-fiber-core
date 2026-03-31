package bulkjobitemmessage

import (
	"context"

	"go-fiber-core/internal/models"

	"gorm.io/gorm"
)

type BulkJobItemMessageWriter interface {
	Create(ctx context.Context, db *gorm.DB, msg *models.BulkJobItemMessage) error
}

type bulkJobItemMessageWriterRepo struct{}

func NewBulkJobItemMessageWriterRepo() BulkJobItemMessageWriter { return &bulkJobItemMessageWriterRepo{} }

func (w *bulkJobItemMessageWriterRepo) Create(ctx context.Context, db *gorm.DB, msg *models.BulkJobItemMessage) error {
	return db.WithContext(ctx).Create(msg).Error
}

