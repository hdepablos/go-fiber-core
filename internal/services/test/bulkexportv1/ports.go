package bulkexportv1

import (
	"context"
	"io"
	"time"

	"go-fiber-core/internal/models"
)

type BulkJobReader interface {
	GetStatus(ctx context.Context, bulkJobID int64) (models.BulkJobStatus, error)
}

type BulkJobWriter interface {
	UpdateStatus(ctx context.Context, bulkJobID int64, status models.BulkJobStatus) error
}

type BulkJobItemReader interface {
	ListIDsAfter(ctx context.Context, bulkJobID int64, lastID int64, limit int) ([]int64, error)
	FindByIDs(ctx context.Context, bulkJobID int64, ids []int64) ([]models.BulkJobItem, error)
}

type BulkJobOutputWriter interface {
	Create(ctx context.Context, output *models.BulkJobOutput) error
}

type Cache interface {
	Del(ctx context.Context, keys ...string) error
	Expire(ctx context.Context, key string, ttl time.Duration) error
	GetBytes(ctx context.Context, key string) ([]byte, error)
	LIndex(ctx context.Context, key string, index int64) (string, error)
	LRange(ctx context.Context, key string, start, stop int64) ([]string, error)
	RPush(ctx context.Context, key string, values ...string) error
	SetBytes(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

type CSVBuilder interface {
	Build(items []models.BulkJobItem, includeHeader bool) ([]byte, error)
}

type CompletedPart struct {
	ETag       string
	PartNumber int32
}

type ObjectInfo struct {
	ContentLength int64
}

type ObjectStore interface {
	EnsureBucket(ctx context.Context, bucket string) error
	PutObject(ctx context.Context, bucket string, key string, body []byte, contentType string) error
	GetObject(ctx context.Context, bucket string, key string) (io.ReadCloser, error)
	HeadObject(ctx context.Context, bucket string, key string) (ObjectInfo, error)
	DeletePrefix(ctx context.Context, bucket string, prefix string) error

	CreateMultipartUpload(ctx context.Context, bucket string, key string, contentType string) (string, error)
	UploadPart(ctx context.Context, bucket string, key string, uploadID string, partNumber int32, body []byte) (string, error)
	CompleteMultipartUpload(ctx context.Context, bucket string, key string, uploadID string, parts []CompletedPart) error
	AbortMultipartUpload(ctx context.Context, bucket string, key string, uploadID string) error
}
