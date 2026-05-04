package batchflow

import "context"

const (
	SourceModeMaterialized = "materialized"
	SourceModeCursor       = "cursor"
)

type CursorRunResult struct {
	Summary       Summary        `json:"summary"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	InitialCursor map[string]any `json:"initial_cursor,omitempty"`
}

type CursorBatchRequest struct {
	BatchSize   int            `json:"batch_size"`
	BatchIndex  int            `json:"batch_index"`
	ShardIndex  int            `json:"shard_index"`
	TotalShards int            `json:"total_shards"`
	Cursor      map[string]any `json:"cursor,omitempty"`
}

type CursorBatchResult struct {
	Batch      Batch          `json:"batch"`
	NextCursor map[string]any `json:"next_cursor,omitempty"`
	HasMore    bool           `json:"has_more"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// CursorDataProvider permite procesar una fuente incremental sin materializar
// todos los batches en Redis durante el start.
type CursorDataProvider interface {
	DataProvider
	PrepareCursorRun(ctx context.Context, execCtx ExecutionContext, batchSize int) (CursorRunResult, error)
	LoadCursorBatch(ctx context.Context, execCtx ExecutionContext, req CursorBatchRequest) (CursorBatchResult, error)
}

func NormalizeSourceMode(raw string) string {
	switch raw {
	case SourceModeCursor:
		return SourceModeCursor
	default:
		return SourceModeMaterialized
	}
}
