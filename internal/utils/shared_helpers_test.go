package utils

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestToIntAndToInt64ConvertCommonInputShapes(t *testing.T) {
	t.Parallel()

	if got := ToInt("42"); got != 42 {
		t.Fatalf("ToInt string = %d, want 42", got)
	}
	if got := ToInt64(float64(99)); got != 99 {
		t.Fatalf("ToInt64 float64 = %d, want 99", got)
	}
	if got := ToInt64(json.Number("1234")); got != 1234 {
		t.Fatalf("ToInt64 json.Number = %d, want 1234", got)
	}
}

func TestBuildCSVLineHonorsSeparatorAndTrimsTrailingNewline(t *testing.T) {
	t.Parallel()

	line, err := BuildCSVLine([]string{"a", "b,c"}, ';')
	if err != nil {
		t.Fatalf("BuildCSVLine returned error: %v", err)
	}
	if line != "a;b,c" {
		t.Fatalf("BuildCSVLine = %q, want %q", line, "a;b,c")
	}
}

func TestFormatDateSupportsConfiguredLayouts(t *testing.T) {
	t.Parallel()

	formatted, err := FormatDate("2026-04-14T13:45:10Z", "DDMMYYYY")
	if err != nil {
		t.Fatalf("FormatDate returned error: %v", err)
	}
	if formatted != "14042026" {
		t.Fatalf("FormatDate = %q, want %q", formatted, "14042026")
	}
}

func TestExtractJSONFieldsSupportsBase64EncodedPayloads(t *testing.T) {
	t.Parallel()

	rawObject := []byte(`{"amount":"100.5","descount1":"1","descount2":"2"}`)
	encoded := base64.StdEncoding.EncodeToString(rawObject)
	rawPayload, err := json.Marshal(encoded)
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	fields, err := ExtractJSONFields(rawPayload, []string{"amount", "descount1", "descount2"})
	if err != nil {
		t.Fatalf("ExtractJSONFields returned error: %v", err)
	}

	expected := []string{"100.5", "1", "2"}
	for i := range expected {
		if fields[i] != expected[i] {
			t.Fatalf("fields[%d] = %q, want %q", i, fields[i], expected[i])
		}
	}
}
