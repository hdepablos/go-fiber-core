package generar_archivo_banco_galicia

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"go-fiber-core/internal/services/exportmanager"
)

func TestHeaderBuilderUsesSemicolonAndProjectedDataColumns(t *testing.T) {
	builder := NewHeaderBuilder()

	lines, err := builder.BuildHeader(context.Background(), exportmanager.ExecutionContext{})
	if err != nil {
		t.Fatalf("BuildHeader() error = %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("BuildHeader() lines = %d, want 1", len(lines))
	}

	want := "id;bulk_job_id;row_number;reference_key;status_code;last_detail_message;created_at;updated_at;amount;descount1;descount2;sweep_days;collection_file_id;new_importe"
	if lines[0] != want {
		t.Fatalf("BuildHeader() line = %q, want %q", lines[0], want)
	}
}

func TestBodyBuilderProjectsSelectedDataFields(t *testing.T) {
	builder := NewBodyBuilder()
	lastDetail := "detalle"

	item, err := json.Marshal(struct {
		ID                int64           `json:"ID"`
		BulkJobID         int64           `json:"BulkJobID"`
		RowNumber         int             `json:"RowNumber"`
		ReferenceKey      string          `json:"ReferenceKey"`
		StatusCode        string          `json:"StatusCode"`
		LastDetailMessage *string         `json:"LastDetailMessage"`
		CreatedAt         time.Time       `json:"CreatedAt"`
		UpdatedAt         time.Time       `json:"UpdatedAt"`
		Data              json.RawMessage `json:"Data"`
	}{
		ID:                10,
		BulkJobID:         20,
		RowNumber:         30,
		ReferenceKey:      "REF-1",
		StatusCode:        "ERROR_PROCESS",
		LastDetailMessage: &lastDetail,
		CreatedAt:         time.Date(2026, 4, 13, 10, 0, 0, 0, time.UTC),
		UpdatedAt:         time.Date(2026, 4, 13, 11, 0, 0, 0, time.UTC),
		Data: json.RawMessage(`{
			"amount": 100.5,
			"descount1": 7,
			"descount2": "9",
			"sweep_days": 12,
			"collection_file_id": 456,
			"ignored_key": "omit"
		}`),
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	lines, err := builder.BuildBodyLines(context.Background(), exportmanager.ExecutionContext{}, item)
	if err != nil {
		t.Fatalf("BuildBodyLines() error = %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("BuildBodyLines() lines = %d, want 1", len(lines))
	}

	want := "10;20;30;REF-1;ERROR_PROCESS;detalle;2026-04-13 07:00:00;13042026;100.5;7;9;12;456;84.5"
	if lines[0] != want {
		t.Fatalf("BuildBodyLines() line = %q, want %q", lines[0], want)
	}
}

func TestBodyBuilderSupportsLegacyBase64EncodedData(t *testing.T) {
	builder := NewBodyBuilder()

	legacyData := base64.StdEncoding.EncodeToString([]byte(`{
		"amount": 100.5,
		"descount1": 7,
		"descount2": "9",
		"sweep_days": 12,
		"collection_file_id": 456
	}`))

	item, err := json.Marshal(struct {
		ID         int64     `json:"ID"`
		BulkJobID  int64     `json:"BulkJobID"`
		RowNumber  int       `json:"RowNumber"`
		StatusCode string    `json:"StatusCode"`
		CreatedAt  time.Time `json:"CreatedAt"`
		UpdatedAt  time.Time `json:"UpdatedAt"`
		Data       string    `json:"Data"`
	}{
		ID:         10,
		BulkJobID:  20,
		RowNumber:  30,
		StatusCode: "ERROR_PROCESS",
		CreatedAt:  time.Date(2026, 4, 13, 10, 0, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2026, 4, 13, 11, 0, 0, 0, time.UTC),
		Data:       legacyData,
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	lines, err := builder.BuildBodyLines(context.Background(), exportmanager.ExecutionContext{}, item)
	if err != nil {
		t.Fatalf("BuildBodyLines() error = %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("BuildBodyLines() lines = %d, want 1", len(lines))
	}

	want := "10;20;30;;ERROR_PROCESS;;2026-04-13 07:00:00;13042026;100.5;7;9;12;456;84.5"
	if lines[0] != want {
		t.Fatalf("BuildBodyLines() line = %q, want %q", lines[0], want)
	}
}
