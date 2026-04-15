package generar_archivo_banco_galicia

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"go-fiber-core/internal/models"
	"go-fiber-core/internal/services/exportmanager"
	"go-fiber-core/internal/utils"
)

// ──────────────────────────────────────────────
// Columnas de exportación
// ──────────────────────────────────────────────

var exportColumns = [5]string{
	"amount",
	"descount1",
	"descount2",
	"sweep_days",
	"collection_file_id",
}

var csvHeader = append(
	[]string{
		"id",
		"bulk_job_id",
		"row_number",
		"reference_key",
		"status_code",
		"last_detail_message",
		"created_at",
		"updated_at",
	},
	append(exportColumns[:], "new_importe")...,
)

// ──────────────────────────────────────────────
// exportData agrupa los campos extraídos del payload Data
// junto con el campo calculado new_importe.
// ──────────────────────────────────────────────

type exportData struct {
	fields     [5]string
	newImporte string
}

// calcNewImporte computa: amount - (descount1 + descount2)
// Cualquier descuento nulo o vacío se trata como 0.
func calcNewImporte(amount, descount1, descount2 string) (string, error) {
	amt, err := utils.ParseDecimal(amount)
	if err != nil {
		return "", fmt.Errorf("parsear amount %q: %w", amount, err)
	}

	d1 := utils.ParseDecimalOrZero(descount1)
	d2 := utils.ParseDecimalOrZero(descount2)

	result := amt - (d1 + d2)
	return strconv.FormatFloat(result, 'f', -1, 64), nil
}

// ──────────────────────────────────────────────
// Tipos internos
// ──────────────────────────────────────────────

type bulkItemPayload struct {
	ID                int64                `json:"ID"`
	BulkJobID         int64                `json:"BulkJobID"`
	RowNumber         int                  `json:"RowNumber"`
	ReferenceKey      string               `json:"ReferenceKey"`
	StatusCode        models.BulkJobStatus `json:"StatusCode"`
	LastDetailMessage *string              `json:"LastDetailMessage"`
	CreatedAt         time.Time            `json:"CreatedAt"`
	UpdatedAt         time.Time            `json:"UpdatedAt"`
	Data              json.RawMessage      `json:"Data"`
}

func (p *bulkItemPayload) lastDetail() string {
	if p.LastDetailMessage != nil {
		return *p.LastDetailMessage
	}
	return ""
}

func (p *bulkItemPayload) toRow(data exportData) ([]string, error) {
	createdAt, err := utils.FormatDate(p.CreatedAt.Format(time.RFC3339), "YYYY-MM-DD HH:mm:ss")
	if err != nil {
		return nil, fmt.Errorf("format createdAt: %w", err)
	}

	updatedAt, err := utils.FormatDate(p.UpdatedAt.Format(time.RFC3339), "DDMMYYYY")
	if err != nil {
		return nil, fmt.Errorf("format updatedAt: %w", err)
	}

	return []string{
		fmt.Sprintf("%d", p.ID),
		fmt.Sprintf("%d", p.BulkJobID),
		fmt.Sprintf("%d", p.RowNumber),
		p.ReferenceKey,
		string(p.StatusCode),
		p.lastDetail(),
		createdAt,
		updatedAt,
		data.fields[0],  // amount
		data.fields[1],  // descount1
		data.fields[2],  // descount2
		data.fields[3],  // sweep_days
		data.fields[4],  // collection_file_id
		data.newImporte, // new_importe
	}, nil
}

// ──────────────────────────────────────────────
// HeaderBuilder
// ──────────────────────────────────────────────

type headerBuilder struct{}

func NewHeaderBuilder() exportmanager.HeaderBuilder { return &headerBuilder{} }

func (b *headerBuilder) BuildHeader(_ context.Context, _ exportmanager.ExecutionContext) ([]string, error) {
	line, err := utils.BuildCSVLine(csvHeader, ';')
	if err != nil {
		return nil, fmt.Errorf("build header: %w", err)
	}
	return []string{line}, nil
}

// ──────────────────────────────────────────────
// BodyBuilder
// ──────────────────────────────────────────────

type bodyBuilder struct{}

func NewBodyBuilder() exportmanager.BodyBuilder { return &bodyBuilder{} }

func (b *bodyBuilder) BuildBodyLines(_ context.Context, _ exportmanager.ExecutionContext, item json.RawMessage) ([]string, error) {
	var payload bulkItemPayload
	if err := json.Unmarshal(item, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal bulk item: %w", err)
	}

	data, err := extractExportData(payload.Data)
	if err != nil {
		return nil, fmt.Errorf("extract export data: %w", err)
	}

	row, err := payload.toRow(data)
	if err != nil {
		return nil, fmt.Errorf("build row: %w", err)
	}

	line, err := utils.BuildCSVLine(row, ';')
	if err != nil {
		return nil, fmt.Errorf("build body line: %w", err)
	}
	return []string{line}, nil
}

// ──────────────────────────────────────────────
// FooterBuilder
// ──────────────────────────────────────────────

type footerBuilder struct{}

func NewFooterBuilder() exportmanager.FooterBuilder { return &footerBuilder{} }

func (b *footerBuilder) BuildFooter(_ context.Context, _ exportmanager.ExecutionContext) ([]string, error) {
	// TODO: personalizar el footer.
	// Si no deseas footer, reemplaza esto por: return []string{}, nil
	line, err := utils.BuildCSVLine([]string{"footer"}, ';')
	if err != nil {
		return nil, fmt.Errorf("build footer: %w", err)
	}
	return []string{line}, nil
}

// ──────────────────────────────────────────────
// Helpers Data
// ──────────────────────────────────────────────

// extractExportData extrae los campos de exportColumns y calcula new_importe.
func extractExportData(raw json.RawMessage) (exportData, error) {
	fields, err := extractDataFields(raw)
	if err != nil {
		return exportData{}, err
	}

	// fields[0]=amount, fields[1]=descount1, fields[2]=descount2
	newImporte, err := calcNewImporte(fields[0], fields[1], fields[2])
	if err != nil {
		return exportData{}, fmt.Errorf("calcular new_importe: %w", err)
	}

	return exportData{
		fields:     fields,
		newImporte: newImporte,
	}, nil
}

func extractDataFields(raw json.RawMessage) ([5]string, error) {
	values, err := utils.ExtractJSONFields(raw, exportColumns[:])
	if err != nil {
		return [5]string{}, err
	}

	var fields [5]string
	copy(fields[:], values)
	return fields, nil
}
