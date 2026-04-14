package exportmanager

import (
	"context"
	"encoding/json"
	"io"
	"time"
)

type Input struct {
	RedisKey string         `json:"key_redis,omitempty"`
	ParentID int64          `json:"id"`
	Filters  any            `json:"filters,omitempty"`
	Values   map[string]any `json:"values,omitempty"`
}

type FilterCondition struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    any    `json:"value"`
}

type RuntimeValues interface {
	Set(ctx context.Context, key string, value any) error
	Get(ctx context.Context, key string, dest any) error
	Delete(ctx context.Context, key string) error
}

type ExecutionContext struct {
	Input   Input
	Summary Summary
	Runtime RuntimeValues
}

type Summary struct {
	TotalRecords int64          `json:"total_records"`
	TotalAmount  float64        `json:"total_amount"`
	Defaults     map[string]any `json:"defaults,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

type Batch struct {
	Items []json.RawMessage `json:"items"`
}

type LoadBatchesResult struct {
	Batches  []Batch        `json:"batches"`
	Summary  Summary        `json:"summary"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type DataProvider interface {
	LoadBatches(ctx context.Context, execCtx ExecutionContext, batchSize int) (LoadBatchesResult, error)
}

type HeaderBuilder interface {
	BuildHeader(ctx context.Context, execCtx ExecutionContext) ([]string, error)
}

type BodyBuilder interface {
	BuildBodyLines(ctx context.Context, execCtx ExecutionContext, item json.RawMessage) ([]string, error)
}

type FooterBuilder interface {
	BuildFooter(ctx context.Context, execCtx ExecutionContext) ([]string, error)
}

type ParentLifecycle interface {
	Start(ctx context.Context, execCtx ExecutionContext) error
	End(ctx context.Context, execCtx ExecutionContext, output OutputResult) error
	Fail(ctx context.Context, execCtx ExecutionContext, cause error) error
}

type OutputRegistrar interface {
	Register(ctx context.Context, execCtx ExecutionContext, output OutputResult) error
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
	DeleteObject(ctx context.Context, bucket string, key string) error
	CreateMultipartUpload(ctx context.Context, bucket string, key string, contentType string) (string, error)
	UploadPart(ctx context.Context, bucket string, key string, uploadID string, partNumber int32, body []byte) (string, error)
	CompleteMultipartUpload(ctx context.Context, bucket string, key string, uploadID string, parts []CompletedPart) error
	AbortMultipartUpload(ctx context.Context, bucket string, key string, uploadID string) error
}

type StartRequest struct {
	Input      Input
	BatchSize  int
	RedisTTL   time.Duration
	S3Bucket   string
	PartPrefix string
}

type StartResult struct {
	RedisKey       string
	BatchesListKey string
	PartsListKey   string
	TotalBatches   int
	Summary        Summary
	Metadata       map[string]any
	S3Bucket       string
	PartPrefix     string
}

type ProcessBatchRequest struct {
	Input          Input
	BatchesListKey string
	PartsListKey   string
	S3Bucket       string
	PartPrefix     string
	BatchIndex     int
	TotalBatches   int
}

type ProcessBatchResult struct {
	NextBatchIndex int
	IsLastBatch    bool
	ProcessedCount int
	S3PartKey      string
}

type FinalizeRequest struct {
	Input        Input
	PartsListKey string
	S3Bucket     string
	FileBase     string
	TotalParts   int
}

type OutputResult struct {
	Bucket      string
	Key         string
	Path        string
	FileSize    int64
	ContentType string
	Parts       int
}

type PreviewComponents struct {
	DataProvider  DataProvider
	HeaderBuilder HeaderBuilder
	BodyBuilder   BodyBuilder
	FooterBuilder FooterBuilder
	StateStore    StateStore
}

type PreviewProvider interface {
	PreviewComponents() PreviewComponents
}

type PreviewRequest struct {
	ProcessTypeID            int64
	ProcessTypeName          string
	ResolvedProcessVersionID int64
	ExecutionKeys            []string
	Mode                     string
	Input                    Input
	BatchSize                int
	Limit                    int
	Offset                   int
	ItemIDs                  []int64
	RowNumbers               []int
}

type PreviewResponse struct {
	ProcessTypeID            int64          `json:"process_type_id"`
	ProcessTypeName          string         `json:"process_type_name,omitempty"`
	ResolvedProcessVersionID int64          `json:"resolved_process_version_id,omitempty"`
	ResolvedExecutionKeys    []string       `json:"resolved_execution_keys,omitempty"`
	Mode                     string         `json:"mode"`
	RedisKey                 string         `json:"redis_key"`
	Summary                  Summary        `json:"summary"`
	AppliedFilters           any            `json:"applied_filters,omitempty"`
	TotalBatches             int            `json:"total_batches"`
	RenderedCount            int            `json:"rendered_count"`
	HeaderLines              []string       `json:"header_lines,omitempty"`
	BodyLines                []string       `json:"body_lines,omitempty"`
	FooterLines              []string       `json:"footer_lines,omitempty"`
	Lines                    []string       `json:"lines,omitempty"`
	Selection                map[string]any `json:"selection,omitempty"`
}
