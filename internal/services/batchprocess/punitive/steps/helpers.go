package steps

import (
	"go-fiber-core/internal/services/serviceconfig/contracts"
	"go-fiber-core/internal/utils"
)

// resolveParallelShards lee la configuracion efectiva de fan-out y devuelve 1 como fallback seguro.
func resolveParallelShards(ctx *contracts.ServiceContext) int {
	if ctx == nil || ctx.CurrentStepConfig == nil {
		return 1
	}
	if v, ok := ctx.CurrentStepConfig["parallel_shards"]; ok {
		if parsed := utils.ToInt(v); parsed > 0 {
			return parsed
		}
	}
	if rawMode, ok := ctx.CurrentStepConfig["execution_mode"].(map[string]any); ok {
		if v, ok := rawMode["parallel_shards"]; ok {
			if parsed := utils.ToInt(v); parsed > 0 {
				return parsed
			}
		}
	}
	return 1
}

// cloneInput evita que cada shard comparta el mismo mapa mutable de input.
func cloneInput(input map[string]any) map[string]any {
	out := make(map[string]any, len(input))
	for k, v := range input {
		out[k] = v
	}
	return out
}
