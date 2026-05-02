package generar_archivo_banco_galicia

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go-fiber-core/internal/models"
	"go-fiber-core/internal/services/exportmanager"
)

// ──────────────────────────────────────────────
// Fecha
// ──────────────────────────────────────────────

var argLocation = time.FixedZone("ART", -3*60*60)

var formatPatterns = map[string]string{
	"YYYY-MM-DD":            "2006-01-02",
	"DDMMYYYY":              "02012006",
	"YYYY-MM-DD HH:mm:ss":   "2006-01-02 15:04:05",
	"HH:mm:ss":              "15:04:05",
	"YYYY-MM-DD HH:mm:ss Z": "2006-01-02 15:04:05 -0700",
}

var inputLayouts = []string{
	time.RFC3339,
	"2006-01-02 15:04:05",
	"2006-01-02",
}

// FormatDate parsea dateStr y lo devuelve con el formato indicado en outputFormat.
// outputFormat acepta: "YYYY-MM-DD", "DDMMYYYY", "YYYY-MM-DD HH:mm:ss", "HH:mm:ss", "YYYY-MM-DD HH:mm:ss Z"
func FormatDate(dateStr, outputFormat string) (string, error) {
	goFormat, ok := formatPatterns[outputFormat]
	if !ok {
		return "", fmt.Errorf("formato de salida no soportado: %q", outputFormat)
	}

	var (
		parsed time.Time
		err    error
	)
	for _, layout := range inputLayouts {
		parsed, err = time.Parse(layout, dateStr)
		if err == nil {
			break
		}
	}
	if err != nil {
		return "", fmt.Errorf("no se pudo parsear la fecha %q: %w", dateStr, err)
	}

	return parsed.In(argLocation).Format(goFormat), nil
}

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
	amt, err := parseDecimal(amount)
	if err != nil {
		return "", fmt.Errorf("parsear amount %q: %w", amount, err)
	}

	d1 := parseDecimalOrZero(descount1)
	d2 := parseDecimalOrZero(descount2)

	result := amt - (d1 + d2)
	return strconv.FormatFloat(result, 'f', -1, 64), nil
}

// parseDecimal convierte un string a float64; devuelve error si está vacío o es inválido.
func parseDecimal(s string) (float64, error) {
	if s == "" {
		return 0, fmt.Errorf("valor vacío")
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("valor no numérico %q: %w", s, err)
	}
	return v, nil
}

// parseDecimalOrZero convierte un string a float64; devuelve 0 si está vacío o es inválido.
func parseDecimalOrZero(s string) float64 {
	if s == "" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
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
	createdAt, err := FormatDate(p.CreatedAt.Format(time.RFC3339), "YYYY-MM-DD HH:mm:ss")
	if err != nil {
		return nil, fmt.Errorf("format createdAt: %w", err)
	}

	updatedAt, err := FormatDate(p.UpdatedAt.Format(time.RFC3339), "DDMMYYYY")
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

type HeaderBuilder struct{}

func NewHeaderBuilder() *HeaderBuilder { return &HeaderBuilder{} }

func (b *HeaderBuilder) BuildHeader(_ context.Context, _ exportmanager.ExecutionContext) ([]string, error) {
	line, err := toCSVLine(csvHeader)
	if err != nil {
		return nil, fmt.Errorf("build header: %w", err)
	}
	return []string{line}, nil
}

// ──────────────────────────────────────────────
// BodyBuilder
// ──────────────────────────────────────────────

type BodyBuilder struct{}

func NewBodyBuilder() *BodyBuilder { return &BodyBuilder{} }

func (b *BodyBuilder) BuildBodyLines(_ context.Context, _ exportmanager.ExecutionContext, item json.RawMessage) ([]string, error) {
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

	line, err := toCSVLine(row)
	if err != nil {
		return nil, fmt.Errorf("build body line: %w", err)
	}
	return []string{line}, nil
}

// ──────────────────────────────────────────────
// FooterBuilder
// ──────────────────────────────────────────────

type FooterBuilder struct{}

func NewFooterBuilder() *FooterBuilder { return &FooterBuilder{} }

func (b *FooterBuilder) BuildFooter(_ context.Context, _ exportmanager.ExecutionContext) ([]string, error) {
	// TODO: personalizar el footer.
	// Si no deseas footer, reemplaza esto por: return []string{}, nil
	line, err := toCSVLine([]string{"footer"})
	if err != nil {
		return nil, fmt.Errorf("build footer: %w", err)
	}
	return []string{line}, nil
}

// ──────────────────────────────────────────────
// Helpers CSV
// ──────────────────────────────────────────────

func toCSVLine(fields []string) (string, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	w.Comma = ';'

	if err := w.Write(fields); err != nil {
		return "", err
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", err
	}

	return strings.TrimRight(buf.String(), "\n"), nil
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
		fields[i] = stringifyValue(data[key])
	}
	return fields, nil
}

func stringifyValue(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case json.Number:
		return v.String()
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case bool:
		return strconv.FormatBool(v)
	case int:
		return strconv.Itoa(v)
	case int8, int16, int32, int64:
		return strconv.FormatInt(toInt64(v), 10)
	case uint, uint8, uint16, uint32, uint64:
		return strconv.FormatUint(toUint64(v), 10)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return string(b)
	}
}

func toInt64(v any) int64 {
	switch n := v.(type) {
	case int8:
		return int64(n)
	case int16:
		return int64(n)
	case int32:
		return int64(n)
	case int64:
		return n
	default:
		return 0
	}
}

func toUint64(v any) uint64 {
	switch n := v.(type) {
	case uint:
		return uint64(n)
	case uint8:
		return uint64(n)
	case uint16:
		return uint64(n)
	case uint32:
		return uint64(n)
	case uint64:
		return n
	default:
		return 0
	}
}
