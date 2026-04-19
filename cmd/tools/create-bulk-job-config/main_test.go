package main

import (
	"encoding/json"
	"testing"
)

func TestBuildDefaultConfig(t *testing.T) {
	raw, err := buildDefaultConfig(options{
		OperatorID:               7,
		ProcessTypeID:            13,
		SedeID:                   0,
		OverrideProcessVersionID: 0,
		Roadmap:                  0,
	})
	if err != nil {
		t.Fatalf("buildDefaultConfig returned error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("config generated is not valid json: %v", err)
	}

	if got["process_type_id"] != float64(13) {
		t.Fatalf("process_type_id = %v, want 13", got["process_type_id"])
	}
	if got["sede_id"] != float64(0) {
		t.Fatalf("sede_id = %v, want 0", got["sede_id"])
	}
	if got["override_process_version_id"] != float64(0) {
		t.Fatalf("override_process_version_id = %v, want 0", got["override_process_version_id"])
	}
	if got["roadmap"] != float64(0) {
		t.Fatalf("roadmap = %v, want 0", got["roadmap"])
	}

	input, ok := got["input"].(map[string]any)
	if !ok {
		t.Fatalf("input has unexpected type: %T", got["input"])
	}
	if len(input) != 0 {
		t.Fatalf("input should be empty, got %v", input)
	}
}
