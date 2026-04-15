package bulkexportv2

import (
	"context"
	"encoding/json"
	"fmt"

	"go-fiber-core/internal/services/exportmanager"
	"go-fiber-core/internal/utils"
)

type HardcodedHeaderBuilder struct{}

func NewHardcodedHeaderBuilder() *HardcodedHeaderBuilder {
	return &HardcodedHeaderBuilder{}
}

func (b *HardcodedHeaderBuilder) BuildHeader(ctx context.Context, execCtx exportmanager.ExecutionContext) ([]string, error) {
	if execCtx.Runtime != nil {
		_ = execCtx.Runtime.Set(ctx, "total_records", execCtx.Summary.TotalRecords)
		_ = execCtx.Runtime.Set(ctx, "total_amount", execCtx.Summary.TotalAmount)
	}
	line, err := utils.BuildCSVLine([]string{"id", "row_number", "reference_key", "data"}, 0)
	if err != nil {
		return nil, err
	}
	return []string{line}, nil
}

type JSONBodyBuilder struct{}

func NewJSONBodyBuilder() *JSONBodyBuilder {
	return &JSONBodyBuilder{}
}

func (b *JSONBodyBuilder) BuildBodyLines(_ context.Context, _ exportmanager.ExecutionContext, item json.RawMessage) ([]string, error) {
	var payload struct {
		ID           int64           `json:"id"`
		RowNumber    int             `json:"row_number"`
		ReferenceKey string          `json:"reference_key"`
		Data         json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(item, &payload); err != nil {
		return nil, err
	}

	line, err := utils.BuildCSVLine([]string{
		fmt.Sprintf("%d", payload.ID),
		fmt.Sprintf("%d", payload.RowNumber),
		payload.ReferenceKey,
		string(payload.Data),
	}, 0)
	if err != nil {
		return nil, err
	}
	return []string{line}, nil
}

type EmptyFooterBuilder struct{}

func NewEmptyFooterBuilder() *EmptyFooterBuilder {
	return &EmptyFooterBuilder{}
}

func (b *EmptyFooterBuilder) BuildFooter(ctx context.Context, execCtx exportmanager.ExecutionContext) ([]string, error) {
	if execCtx.Runtime != nil {
		var totalAmount float64
		if err := execCtx.Runtime.Get(ctx, "total_amount", &totalAmount); err == nil {
			_ = totalAmount
		}
	}
	return []string{}, nil
}
