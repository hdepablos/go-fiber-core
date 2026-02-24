package contracts

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServiceContext_GetAll(t *testing.T) {
	ctx := context.Background()
	svcCtx := NewServiceContextWithCtx(ctx, 30, 5000)

	// Add simple result
	svcCtx.SetResult("step1", StepResult{
		Status: "success",
		Data: map[string]any{
			"score":    85,
			"approved": true,
		},
	})

	// Add another result
	svcCtx.SetResult("step2", StepResult{
		Status: "success",
		Data: map[string]any{
			"extra_info": "verified",
		},
	})

	// Add non-StepResult (fallback case, though SetResult enforces StepResult)
	// But Results is map[string]any, so we can manually inject if needed for test
	svcCtx.mu.Lock()
	svcCtx.Results["manual"] = "raw_value"
	svcCtx.mu.Unlock()

	// GetAll
	all := svcCtx.GetAll()

	assert.Equal(t, 85, all["score"])
	assert.Equal(t, true, all["approved"])
	assert.Equal(t, "verified", all["extra_info"])
	assert.Equal(t, "raw_value", all["manual"])
}
