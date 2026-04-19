package batchflow

import (
	"context"
	"encoding/json"
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

type BatchProcessor interface {
	ProcessBatch(ctx context.Context, execCtx ExecutionContext, batch Batch) (ProcessBatchResult, error)
}

type BatchPreviewer interface {
	PreviewBatch(ctx context.Context, execCtx ExecutionContext, batch Batch) (PreviewBatchResult, error)
}

type ParentLifecycle interface {
	Start(ctx context.Context, execCtx ExecutionContext) error
	End(ctx context.Context, execCtx ExecutionContext, result FinalizeResult) error
	Fail(ctx context.Context, execCtx ExecutionContext, cause error) error
}

type BatchProgressRefresher interface {
	RefreshProgress(ctx context.Context, execCtx ExecutionContext, batch Batch) error
}

type Finalizer interface {
	Finalize(ctx context.Context, execCtx ExecutionContext, req FinalizeRequest) (FinalizeResult, error)
}

type Cache interface {
	Del(ctx context.Context, keys ...string) error
	Expire(ctx context.Context, key string, ttl time.Duration) error
	GetBytes(ctx context.Context, key string) ([]byte, error)
	GetString(ctx context.Context, key string) (string, error)
	IncrBy(ctx context.Context, key string, delta int64) (int64, error)
	LIndex(ctx context.Context, key string, index int64) (string, error)
	LRange(ctx context.Context, key string, start, stop int64) ([]string, error)
	RPush(ctx context.Context, key string, values ...string) error
	SetBytes(ctx context.Context, key string, value []byte, ttl time.Duration) error
	SetString(ctx context.Context, key string, value string, ttl time.Duration) error
	SetNXString(ctx context.Context, key string, value string, ttl time.Duration) (bool, error)
	TTL(ctx context.Context, key string) (time.Duration, error)
}

type StateStore interface {
	Initialize(ctx context.Context, input Input, batches []Batch, summary Summary, metadata map[string]any, ttl time.Duration) (batchesListKey string, err error)
	LoadSummary(ctx context.Context, input Input) (Summary, error)
	LoadBatch(ctx context.Context, batchesListKey string, batchIndex int) (Batch, error)
	Cleanup(ctx context.Context, input Input, batchesListKey string) error
	SetCounter(ctx context.Context, key string, value int64, ttl time.Duration) error
	IncrCounter(ctx context.Context, key string, delta int64, ttl time.Duration) (int64, error)
	GetCounter(ctx context.Context, key string) (int64, error)
	RegisterShards(ctx context.Context, input Input, totalShards int, ttl time.Duration) error
	CompleteShard(ctx context.Context, input Input, shardIndex int, totalShards int, ttl time.Duration) (ShardCompletion, error)
}

type StartRequest struct {
	Input     Input
	BatchSize int
	RedisTTL  time.Duration
}

type StartResult struct {
	RedisKey       string
	BatchesListKey string
	TotalBatches   int
	Summary        Summary
	Metadata       map[string]any
}

type ProcessRequest struct {
	Input             Input
	BatchesListKey    string
	BatchIndex        int
	TotalBatches      int
	ConcurrentBatches int
	ShardIndex        int
	TotalShards       int
	DispatchPacing    DispatchPacingConfig
}

type ProcessBatchResult struct {
	ProcessedCount int            `json:"processed_count"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type ProcessResult struct {
	NextBatchIndex         int            `json:"next_batch_index"`
	IsLastBatch            bool           `json:"is_last_batch"`
	IsShardComplete        bool           `json:"is_shard_complete"`
	ProcessedCount         int            `json:"processed_count"`
	BatchesProcessed       int            `json:"batches_processed"`
	ShardIndex             int            `json:"shard_index,omitempty"`
	TotalShards            int            `json:"total_shards,omitempty"`
	CompletedShards        int64          `json:"completed_shards,omitempty"`
	ShouldDispatchNextStep bool           `json:"should_dispatch_next_step,omitempty"`
	Metadata               map[string]any `json:"metadata,omitempty"`
}

type DispatchRequest struct {
	Input          Input
	TotalBatches   int
	ParallelShards int
}

type DispatchResult struct {
	TotalShards         int   `json:"total_shards"`
	InitialBatchIndexes []int `json:"initial_batch_indexes"`
}

type ShardCompletion struct {
	CompletedShards int64 `json:"completed_shards"`
	ShouldFinalize  bool  `json:"should_finalize"`
}

type FinalizeRequest struct {
	Input          Input
	BatchesListKey string
	TotalBatches   int
}

type FinalizeResult struct {
	Summary  Summary        `json:"summary"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type Manager interface {
	Start(ctx context.Context, req StartRequest) (StartResult, error)
	DispatchShards(ctx context.Context, req DispatchRequest) (DispatchResult, error)
	ProcessBatch(ctx context.Context, req ProcessRequest) (ProcessResult, error)
	Finalize(ctx context.Context, req FinalizeRequest) (FinalizeResult, error)
	Fail(ctx context.Context, input Input, cause error) error
}

type PreviewComponents struct {
	DataProvider      DataProvider
	BatchProcessor    BatchProcessor
	BatchPreviewer    BatchPreviewer
	ProgressRefresher BatchProgressRefresher
	StateStore        StateStore
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
	ApplyChanges             bool
	DispatchPacing           DispatchPacingConfig
	Input                    Input
	BatchSize                int
	Limit                    int
	Offset                   int
	BatchIndex               *int
	ItemIDs                  []int64
	RowNumbers               []int
}

type PreviewMessage struct {
	Severity      string         `json:"severity"`
	Code          string         `json:"code,omitempty"`
	DetailMessage string         `json:"detail_message"`
	Meta          map[string]any `json:"meta,omitempty"`
}

type PreviewItemResult struct {
	ItemID       int64            `json:"item_id,omitempty"`
	RowNumber    int              `json:"row_number,omitempty"`
	ReferenceKey string           `json:"reference_key,omitempty"`
	Status       string           `json:"status"`
	Message      string           `json:"message,omitempty"`
	Messages     []PreviewMessage `json:"messages,omitempty"`
	Metadata     map[string]any   `json:"metadata,omitempty"`
}

type PreviewBatchResult struct {
	Items          []PreviewItemResult `json:"items"`
	ProcessedCount int                 `json:"processed_count"`
	Metadata       map[string]any      `json:"metadata,omitempty"`
}

type PreviewResponse struct {
	ProcessTypeID            int64               `json:"process_type_id"`
	ProcessTypeName          string              `json:"process_type_name,omitempty"`
	ResolvedProcessVersionID int64               `json:"resolved_process_version_id,omitempty"`
	ResolvedExecutionKeys    []string            `json:"resolved_execution_keys,omitempty"`
	Mode                     string              `json:"mode"`
	AppliedChanges           bool                `json:"applied_changes,omitempty"`
	ApplyChangesMetadata     map[string]any      `json:"apply_changes_metadata,omitempty"`
	RedisKey                 string              `json:"redis_key"`
	Summary                  Summary             `json:"summary"`
	AppliedFilters           any                 `json:"applied_filters,omitempty"`
	TotalBatches             int                 `json:"total_batches"`
	BatchIndex               *int                `json:"batch_index,omitempty"`
	RenderedCount            int                 `json:"rendered_count"`
	Items                    []PreviewItemResult `json:"items,omitempty"`
	Selection                map[string]any      `json:"selection,omitempty"`
}
