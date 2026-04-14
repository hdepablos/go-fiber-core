package exportmanager

import (
	"fmt"
	"strings"
)

// NormalizeFilters accepts the legacy map format and the new array-of-objects format.
func NormalizeFilters(raw any) (map[string]any, error) {
	if raw == nil {
		return nil, nil
	}

	switch typed := raw.(type) {
	case map[string]any:
		return typed, nil
	case []FilterCondition:
		return normalizeFilterConditions(typed)
	case []any:
		conditions := make([]FilterCondition, 0, len(typed))
		for _, entry := range typed {
			m, ok := entry.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("filters inválidos: cada filtro debe ser un objeto")
			}
			field, _ := m["field"].(string)
			operator, _ := m["operator"].(string)
			conditions = append(conditions, FilterCondition{
				Field:    strings.TrimSpace(field),
				Operator: strings.TrimSpace(operator),
				Value:    m["value"],
			})
		}
		return normalizeFilterConditions(conditions)
	default:
		return nil, fmt.Errorf("filters inválidos: formato no soportado")
	}
}

func normalizeFilterConditions(filters []FilterCondition) (map[string]any, error) {
	out := make(map[string]any, len(filters))
	for _, filter := range filters {
		field := strings.TrimSpace(filter.Field)
		if field == "" {
			return nil, fmt.Errorf("filters inválidos: field es requerido")
		}

		operator := strings.ToLower(strings.TrimSpace(filter.Operator))
		if operator == "" {
			operator = "eq"
		}

		switch operator {
		case "eq":
			out[field] = filter.Value
		case "in":
			values, ok := toAnySlice(filter.Value)
			if !ok {
				return nil, fmt.Errorf("filters inválidos: operator 'in' requiere un arreglo para %s", field)
			}
			out[field] = values
		default:
			return nil, fmt.Errorf("filters inválidos: operator %q no soportado", operator)
		}
	}
	return out, nil
}

func toAnySlice(value any) ([]any, bool) {
	switch typed := value.(type) {
	case []any:
		return typed, true
	case []string:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, item)
		}
		return out, true
	case []int:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, item)
		}
		return out, true
	case []int64:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, item)
		}
		return out, true
	case []float64:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, item)
		}
		return out, true
	default:
		return nil, false
	}
}
