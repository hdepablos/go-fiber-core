package generar_archivo_banco_galicia

import (
	"context"
	"testing"

	"go-fiber-core/internal/services/serviceconfig/contracts"
)

func TestBuildStartInputWithoutFiltersKeepsNil(t *testing.T) {
	ctx := contracts.NewServiceContextFromInput(context.Background(), map[string]any{
		"id": 2,
	})

	input, err := buildStartInput(ctx, nil)
	if err != nil {
		t.Fatalf("buildStartInput() error = %v", err)
	}

	if input.Filters != nil {
		t.Fatalf("Filters = %#v, want nil", input.Filters)
	}
}

func TestBuildStartInputMergesConfigAndInputFilters(t *testing.T) {
	ctx := contracts.NewServiceContextFromInput(context.Background(), map[string]any{
		"id": 2,
		"filters": []any{
			map[string]any{"field": "status_code", "operator": "eq", "value": "ERROR_PROCESS"},
			map[string]any{"field": "row_number", "operator": "eq", "value": 10},
		},
	})

	input, err := buildStartInput(ctx, map[string]any{
		"reference_key": "ABC123",
		"status_code":   "IMPORTED",
	})
	if err != nil {
		t.Fatalf("buildStartInput() error = %v", err)
	}

	filters, ok := input.Filters.(map[string]any)
	if !ok {
		t.Fatalf("Filters type = %T, want map[string]any", input.Filters)
	}
	if got := filters["reference_key"]; got != "ABC123" {
		t.Fatalf("reference_key = %v, want ABC123", got)
	}
	if got := filters["status_code"]; got != "ERROR_PROCESS" {
		t.Fatalf("status_code = %v, want ERROR_PROCESS", got)
	}
	if got := filters["row_number"]; got != 10 {
		t.Fatalf("row_number = %v, want 10", got)
	}
}
