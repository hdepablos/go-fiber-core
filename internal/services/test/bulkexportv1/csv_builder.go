package bulkexportv1

import (
	"bytes"
	"encoding/csv"
	"strconv"

	"go-fiber-core/internal/models"
)

type defaultCSVBuilder struct{}

func NewDefaultCSVBuilder() CSVBuilder {
	return &defaultCSVBuilder{}
}

func (b *defaultCSVBuilder) Build(items []models.BulkJobItem, includeHeader bool) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	w.Comma = ','
	if includeHeader {
		if err := w.Write([]string{"id", "row_number", "reference_key", "data"}); err != nil {
			return nil, err
		}
	}
	for _, it := range items {
		rec := []string{
			strconv.FormatInt(it.ID, 10),
			strconv.Itoa(it.RowNumber),
			it.ReferenceKey,
			string(it.Data),
		}
		if err := w.Write(rec); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
