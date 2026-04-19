package main

import (
	"encoding/json"
	"testing"

	"go-fiber-core/internal/services/processlifecycle"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateOptions_AllowsPlainCloneWithoutPacing(t *testing.T) {
	err := validateOptions(options{
		ConfigPath:      "internal/appconfig/config.yml",
		SourceVersionID: 19,
		OperatorID:      1,
		WithPacing:      false,
	})
	require.NoError(t, err)
}

func TestValidateOptions_RequiresPacingValuesWhenEnabled(t *testing.T) {
	err := validateOptions(options{
		ConfigPath:      "internal/appconfig/config.yml",
		SourceVersionID: 19,
		OperatorID:      1,
		WithPacing:      true,
		PacingMessages:  0,
		PacingInterval:  2,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pacing_messages")
}

func TestPatchProcessBatchConfig_AddsDispatchPacingAndDelay(t *testing.T) {
	raw := json.RawMessage(`{
	  "concurrent_batches": 1,
	  "execution_mode": {
	    "type": "sequential"
	  },
	  "execution_policy": {
	    "mode": "ASYNC",
	    "auto_invoke": {
	      "enabled": true,
	      "cursor_field": "batch_index",
	      "stop_condition": "is_last_batch"
	    }
	  }
	}`)

	updated, err := patchProcessBatchConfig(raw, 100, 2)
	require.NoError(t, err)

	var cfg map[string]any
	require.NoError(t, json.Unmarshal(updated, &cfg))

	dispatch := cfg["dispatch_pacing"].(map[string]any)
	assert.Equal(t, true, dispatch["enabled"])
	assert.Equal(t, float64(100), dispatch["messages_per_interval"])
	assert.Equal(t, float64(2), dispatch["interval_seconds"])

	policy := cfg["execution_policy"].(map[string]any)
	autoInvoke := policy["auto_invoke"].(map[string]any)
	assert.Equal(t, "ASYNC", policy["mode"])
	assert.Equal(t, true, autoInvoke["enabled"])
	assert.Equal(t, "batch_index", autoInvoke["cursor_field"])
	assert.Equal(t, "is_last_batch", autoInvoke["stop_condition"])
	assert.Equal(t, float64(2), autoInvoke["delay_seconds"])
}

func TestSelectProcessBatchStep_DetectsSingleProcessBatch(t *testing.T) {
	step, err := selectProcessBatchStep([]processlifecycle.Step{
		{ExecutionKey: "punitorios/start"},
		{ExecutionKey: "punitorios/process_batch"},
		{ExecutionKey: "punitorios/finalize"},
	}, "")
	require.NoError(t, err)
	assert.Equal(t, "punitorios/process_batch", step.ExecutionKey)
}
