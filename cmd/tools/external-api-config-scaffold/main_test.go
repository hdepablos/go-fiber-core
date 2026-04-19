package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpsertAPIBlock_AppendsInsideAPIsSection(t *testing.T) {
	input := "apis:\n  discord:\n    url: ${NOTIFICATION_API_URL}\n    token: ${NOTIFICATION_API_TOKEN}\n    timeout_seconds: 10\n"

	output, err := upsertAPIBlock(input, options{APIKey: "customer_api"})

	require.NoError(t, err)
	assert.Contains(t, output, "  customer_api:\n")
	assert.Contains(t, output, "    url: ${CUSTOMER_API_URL}\n")
	assert.Contains(t, output, "    token: ${CUSTOMER_API_TOKEN}\n")
	assert.Contains(t, output, "    timeout_seconds: 10\n")
}

func TestUpsertAPIBlock_RejectsDuplicateWithoutForce(t *testing.T) {
	input := "apis:\n  customer_api:\n    url: ${CUSTOMER_API_URL}\n    token: ${CUSTOMER_API_TOKEN}\n    timeout_seconds: 10\n"

	_, err := upsertAPIBlock(input, options{APIKey: "customer_api"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ya existe")
}

func TestUpsertAPIBlock_ReplacesDuplicateWithForce(t *testing.T) {
	input := "apis:\n  customer_api:\n    url: ${OLD_URL}\n    token: ${OLD_TOKEN}\n    timeout_seconds: 5\n"

	output, err := upsertAPIBlock(input, options{APIKey: "customer_api", Force: true})

	require.NoError(t, err)
	assert.NotContains(t, output, "${OLD_URL}")
	assert.Equal(t, 1, strings.Count(output, "  customer_api:\n"))
	assert.Contains(t, output, "    timeout_seconds: 10")
}
