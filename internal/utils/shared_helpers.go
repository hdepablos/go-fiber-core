package utils

import (
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go-fiber-core/internal/services/exportmanager"
	"go-fiber-core/internal/services/serviceconfig/contracts"

	"gorm.io/gorm"
)

var sharedDateLocation = time.FixedZone("ART", -3*60*60)

var sharedDateFormats = map[string]string{
	"YYYY-MM-DD":            "2006-01-02",
	"DDMMYYYY":              "02012006",
	"YYYY-MM-DD HH:mm:ss":   "2006-01-02 15:04:05",
	"HH:mm:ss":              "15:04:05",
	"YYYY-MM-DD HH:mm:ss Z": "2006-01-02 15:04:05 -0700",
}

var sharedDateInputLayouts = []string{
	time.RFC3339,
	"2006-01-02 15:04:05",
	"2006-01-02",
}

type BulkJobItemFilterResult struct {
	Query               *gorm.DB
	StatusFilterApplied bool
}

func MustGetInputValue(ctx *contracts.ServiceContext, key string) any {
	value, _ := ctx.GetInputValue(key)
	return value
}

func GetInputValueOrDefault(ctx *contracts.ServiceContext, key string, defaultValue any) any {
	if value, ok := ctx.GetInputValue(key); ok {
		return value
	}
	return defaultValue
}

func ToInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int8:
		return int(typed)
	case int16:
		return int(typed)
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case uint:
		return int(typed)
	case uint8:
		return int(typed)
	case uint16:
		return int(typed)
	case uint32:
		return int(typed)
	case uint64:
		return int(typed)
	case float64:
		return int(typed)
	case float32:
		return int(typed)
	case json.Number:
		n, _ := typed.Int64()
		return int(n)
	case string:
		n, _ := strconv.Atoi(typed)
		return n
	default:
		return 0
	}
}

func ToInt64(value any) int64 {
	switch typed := value.(type) {
	case int:
		return int64(typed)
	case int8:
		return int64(typed)
	case int16:
		return int64(typed)
	case int32:
		return int64(typed)
	case int64:
		return typed
	case uint:
		return int64(typed)
	case uint8:
		return int64(typed)
	case uint16:
		return int64(typed)
	case uint32:
		return int64(typed)
	case uint64:
		return int64(typed)
	case float64:
		return int64(typed)
	case float32:
		return int64(typed)
	case json.Number:
		n, _ := typed.Int64()
		return n
	case string:
		n, _ := strconv.ParseInt(typed, 10, 64)
		return n
	default:
		return 0
	}
}

func ToUint64(value any) uint64 {
	switch typed := value.(type) {
	case uint:
		return uint64(typed)
	case uint8:
		return uint64(typed)
	case uint16:
		return uint64(typed)
	case uint32:
		return uint64(typed)
	case uint64:
		return typed
	case int:
		return uint64(typed)
	case int8:
		return uint64(typed)
	case int16:
		return uint64(typed)
	case int32:
		return uint64(typed)
	case int64:
		return uint64(typed)
	case float64:
		return uint64(typed)
	case float32:
		return uint64(typed)
	case json.Number:
		n, _ := strconv.ParseUint(typed.String(), 10, 64)
		return n
	case string:
		n, _ := strconv.ParseUint(typed, 10, 64)
		return n
	default:
		return 0
	}
}

func BuildCSVLine(fields []string, comma rune) (string, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	if comma != 0 {
		writer.Comma = comma
	}

	if err := writer.Write(fields); err != nil {
		return "", err
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}

	return strings.TrimRight(buf.String(), "\n"), nil
}

func FormatDate(dateStr, outputFormat string) (string, error) {
	goFormat, ok := sharedDateFormats[outputFormat]
	if !ok {
		return "", fmt.Errorf("formato de salida no soportado: %q", outputFormat)
	}

	var (
		parsed time.Time
		err    error
	)
	for _, layout := range sharedDateInputLayouts {
		parsed, err = time.Parse(layout, dateStr)
		if err == nil {
			break
		}
	}
	if err != nil {
		return "", fmt.Errorf("no se pudo parsear la fecha %q: %w", dateStr, err)
	}

	return parsed.In(sharedDateLocation).Format(goFormat), nil
}

func ParseDecimal(value string) (float64, error) {
	if value == "" {
		return 0, fmt.Errorf("valor vacío")
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("valor no numérico %q: %w", value, err)
	}
	return parsed, nil
}

func ParseDecimalOrZero(value string) float64 {
	if value == "" {
		return 0
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func StringifyValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case json.Number:
		return typed.String()
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(typed), 'f', -1, 32)
	case bool:
		return strconv.FormatBool(typed)
	case int:
		return strconv.Itoa(typed)
	case int8, int16, int32, int64:
		return strconv.FormatInt(ToInt64(typed), 10)
	case uint, uint8, uint16, uint32, uint64:
		return strconv.FormatUint(ToUint64(typed), 10)
	default:
		raw, err := json.Marshal(typed)
		if err != nil {
			return ""
		}
		return string(raw)
	}
}

func ExtractAmount(raw []byte) float64 {
	return ExtractNumericField(raw, "amount")
}

func ExtractNumericField(raw []byte, field string) float64 {
	if len(raw) == 0 {
		return 0
	}

	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return 0
	}

	value, ok := data[field]
	if !ok {
		return 0
	}

	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case json.Number:
		parsed, _ := typed.Float64()
		return parsed
	case string:
		parsed, _ := strconv.ParseFloat(typed, 64)
		return parsed
	default:
		return 0
	}
}

func ApplyBulkJobItemFilters(query *gorm.DB, rawFilters any) (BulkJobItemFilterResult, error) {
	filters, err := exportmanager.NormalizeFilters(rawFilters)
	if err != nil {
		return BulkJobItemFilterResult{}, err
	}
	if len(filters) == 0 {
		return BulkJobItemFilterResult{Query: query}, nil
	}

	statusFilterApplied := false
	for key, value := range filters {
		switch key {
		case "bulk_job_items.status_code", "status_code":
			query = ApplyStringFilter(query, "status_code", value)
			statusFilterApplied = true
		case "bulk_job_items.reference_key", "reference_key":
			query = ApplyStringFilter(query, "reference_key", value)
		case "bulk_job_items.row_number", "row_number":
			query = ApplyIntFilter(query, "row_number", value)
		case "bulk_job_items.id", "id":
			query = ApplyInt64Filter(query, "id", value)
		case "bulk_job_items.bulk_job_id", "bulk_job_id":
			query = ApplyInt64Filter(query, "bulk_job_id", value)
		}
	}

	return BulkJobItemFilterResult{
		Query:               query,
		StatusFilterApplied: statusFilterApplied,
	}, nil
}

func ApplyStringFilter(query *gorm.DB, field string, value any) *gorm.DB {
	switch typed := value.(type) {
	case string:
		if typed != "" {
			return query.Where(field+" = ?", typed)
		}
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			if str, ok := item.(string); ok && str != "" {
				values = append(values, str)
			}
		}
		if len(values) > 0 {
			return query.Where(field+" IN ?", values)
		}
	}
	return query
}

func ApplyIntFilter(query *gorm.DB, field string, value any) *gorm.DB {
	switch typed := value.(type) {
	case int:
		return query.Where(field+" = ?", typed)
	case int64:
		return query.Where(field+" = ?", typed)
	case float64:
		return query.Where(field+" = ?", int(typed))
	case string:
		if parsed, err := strconv.Atoi(typed); err == nil {
			return query.Where(field+" = ?", parsed)
		}
	case []any:
		values := make([]int, 0, len(typed))
		for _, item := range typed {
			if parsed, ok := parseIntValue(item); ok {
				values = append(values, parsed)
			}
		}
		if len(values) > 0 {
			return query.Where(field+" IN ?", values)
		}
	}
	return query
}

func ApplyInt64Filter(query *gorm.DB, field string, value any) *gorm.DB {
	switch typed := value.(type) {
	case int:
		return query.Where(field+" = ?", int64(typed))
	case int64:
		return query.Where(field+" = ?", typed)
	case float64:
		return query.Where(field+" = ?", int64(typed))
	case string:
		if parsed, err := strconv.ParseInt(typed, 10, 64); err == nil {
			return query.Where(field+" = ?", parsed)
		}
	case []any:
		values := make([]int64, 0, len(typed))
		for _, item := range typed {
			if parsed, ok := parseInt64Value(item); ok {
				values = append(values, parsed)
			}
		}
		if len(values) > 0 {
			return query.Where(field+" IN ?", values)
		}
	}
	return query
}

func ExtractJSONFields(raw json.RawMessage, keys []string) ([]string, error) {
	fields := make([]string, len(keys))
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
			return ExtractJSONFields(json.RawMessage(decoded), keys)
		}
		return ExtractJSONFields(json.RawMessage(encoded), keys)
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()

	var data map[string]any
	if err := decoder.Decode(&data); err != nil {
		return fields, fmt.Errorf("decode data object: %w", err)
	}

	for i, key := range keys {
		fields[i] = StringifyValue(data[key])
	}
	return fields, nil
}

func parseIntValue(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	case string:
		parsed, err := strconv.Atoi(typed)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func parseInt64Value(value any) (int64, bool) {
	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int64:
		return typed, true
	case float64:
		return int64(typed), true
	case string:
		parsed, err := strconv.ParseInt(typed, 10, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}
