package steps

import (
	"context"
	"errors"

	"go-fiber-core/internal/domain"
	serviceRuntime "go-fiber-core/internal/services/batchprocess/punitive/runtime"
	"go-fiber-core/internal/services/batchflow"
)

// markFailure centraliza el fallback de error para que todos los steps cambien el status del padre igual.
func markFailure(prov serviceRuntime.Provider, ctx context.Context, input batchflow.Input, err error) {
	if errors.Is(err, domain.ErrBusinessRuleViolation) || errors.Is(err, domain.ErrInvalidArgument) {
		return
	}
	_ = prov.Manager().Fail(ctx, input, err)
}
