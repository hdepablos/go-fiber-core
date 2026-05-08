package layout

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"go-fiber-core/internal/models"
	"go-fiber-core/internal/utils"
)

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

type exportData struct {
	fields     [5]string
	newImporte string
}

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

func buildHeaderLine(_ context.Context) (string, error) {
	return utils.BuildCSVLine(csvHeader, ';')
}

func buildBodyLine(_ context.Context, item json.RawMessage) (string, error) {
	var payload bulkItemPayload
	if err := json.Unmarshal(item, &payload); err != nil {
		return "", fmt.Errorf("unmarshal bulk item: %w", err)
	}

	data, err := extractExportData(payload.Data)
	if err != nil {
		return "", fmt.Errorf("extract export data: %w", err)
	}

	row, err := payload.toRow(data)
	if err != nil {
		return "", fmt.Errorf("build row: %w", err)
	}

	line, err := utils.BuildCSVLine(row, ';')
	if err != nil {
		return "", fmt.Errorf("build body line: %w", err)
	}
	return line, nil
}

func buildFooterLine(_ context.Context) (string, error) {
	return utils.BuildCSVLine([]string{"footer"}, ';')
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
		data.fields[0],
		data.fields[1],
		data.fields[2],
		data.fields[3],
		data.fields[4],
		data.newImporte,
	}, nil
}

func extractExportData(raw json.RawMessage) (exportData, error) {
	fields, err := extractDataFields(raw)
	if err != nil {
		return exportData{}, err
	}

	newImporte, err := calcNewImporte(fields[0], fields[1], fields[2])
	if err != nil {
		return exportData{}, fmt.Errorf("calcular new_importe: %w", err)
	}

	return exportData{
		fields:     fields,
		newImporte: newImporte,
	}, nil
}

func calcNewImporte(amount, descount1, descount2 string) (string, error) {
	amt, err := utils.ParseDecimal(amount)
	if err != nil {
		return "", fmt.Errorf("parsear amount %q: %w", amount, err)
	}

	result := amt - (utils.ParseDecimalOrZero(descount1) + utils.ParseDecimalOrZero(descount2))
	return strconv.FormatFloat(result, 'f', -1, 64), nil
}

func extractDataFields(raw json.RawMessage) ([5]string, error) {
	var fields [5]string

	if len(raw) == 0 || string(raw) == "null" {
		return fields, nil
	}

	if raw[0] == '"' {
		var encoded string
		if err := json.Unmarshal(raw, &encoded); err != nil {
			return fields, fmt.Errorf("unmarshal string data: %w", err)
		}
		if encoded == "" {
			return fields, nil
		}
		if decoded, err := base64.StdEncoding.DecodeString(encoded); err == nil && len(decoded) > 0 {
			return extractDataFields(json.RawMessage(decoded))
		}
		return extractDataFields(json.RawMessage(encoded))
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()

	var data map[string]any
	if err := dec.Decode(&data); err != nil {
		return fields, fmt.Errorf("decode data object: %w", err)
	}

	for i, key := range exportColumns {
		fields[i] = utils.StringifyValue(data[key])
	}
	return fields, nil
}
