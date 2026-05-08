package steps

import (
	"fmt"

	"go-fiber-core/internal/services/exportmanager"
	"go-fiber-core/internal/services/serviceconfig/contracts"
	"go-fiber-core/internal/utils"
)

// buildInput reconstruye el input comun que usan process_batch y finalize desde el contexto runtime.
func buildInput(ctx *contracts.ServiceContext) (exportmanager.Input, error) {
	input := exportmanager.Input{
		RedisKey: fmt.Sprint(utils.MustGetInputValue(ctx, "key_redis")),
		ParentID: utils.ToInt64(utils.MustGetInputValue(ctx, "id")),
	}
	if input.ParentID <= 0 {
		return exportmanager.Input{}, fmt.Errorf("id invalido")
	}
	if input.RedisKey == "" {
		return exportmanager.Input{}, fmt.Errorf("key_redis invalida")
	}
	if rawFilters, ok := ctx.GetInputValue("filters"); ok {
		if filters, ok := rawFilters.(map[string]any); ok {
			input.Filters = filters
		}
	}
	return input, nil
}

// buildStartInput arma el input inicial del run antes de que exista key_redis obligatoria.
func buildStartInput(ctx *contracts.ServiceContext) (exportmanager.Input, error) {
	input := exportmanager.Input{
		RedisKey: fmt.Sprint(utils.GetInputValueOrDefault(ctx, "key_redis", "")),
		ParentID: utils.ToInt64(utils.MustGetInputValue(ctx, "id")),
	}
	if input.ParentID <= 0 {
		return exportmanager.Input{}, fmt.Errorf("id invalido")
	}
	if rawFilters, ok := ctx.GetInputValue("filters"); ok {
		if filters, ok := rawFilters.(map[string]any); ok {
			input.Filters = filters
		}
	}
	return input, nil
}
