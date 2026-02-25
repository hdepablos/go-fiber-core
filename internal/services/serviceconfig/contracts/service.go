package contracts

import (
	"context"
	"sync"
)

type StepResult struct {
	Status    string         `json:"status"`
	Message   string         `json:"message,omitempty"`
	Input     map[string]any `json:"input,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
	StepOrder int            `json:"step_order,omitempty"`
}

type ServiceContext struct {
	Ctx               context.Context `json:"-"`
	mu                sync.Mutex
	Input             map[string]any    `json:"input,omitempty"`
	Results           map[string]any    `json:"results"`
	CurrentStepConfig map[string]any    `json:"-"`
	Metrics           *ExecutionMetrics `json:"performance,omitempty"` // Métricas de ejecución (solo en modo test)
}

type ExecutionMetrics struct {
	ExecutionID         string  `json:"execution_id,omitempty"`
	ProcessedItemsCount int     `json:"processed_items_count,omitempty"`
	TotalDurationMs     int64   `json:"total_duration_ms"`
	DBReadMs            int64   `json:"db_read_ms"`
	DBWriteMs           int64   `json:"db_write_ms"`
	DBTotalQueries      int     `json:"db_total_queries"`
	MemoryUsedMB        float64 `json:"memory_used_mb"`
	GoroutinesCount     int     `json:"goroutines"`
}

func NewServiceContext(age, salary int) *ServiceContext {
	return NewServiceContextWithCtx(context.Background(), age, salary)
}

func NewServiceContextWithCtx(ctx context.Context, age, salary int) *ServiceContext {
	if ctx == nil {
		ctx = context.Background()
	}
	return &ServiceContext{
		Ctx:     ctx,
		Input:   map[string]any{"age": age, "salary": salary},
		Results: make(map[string]any),
	}
}

func NewServiceContextFromInput(ctx context.Context, input map[string]any) *ServiceContext {
	if ctx == nil {
		ctx = context.Background()
	}
	m := make(map[string]any)
	for k, v := range input {
		m[k] = v
	}
	return &ServiceContext{
		Ctx:     ctx,
		Input:   m,
		Results: make(map[string]any),
	}
}

func (c *ServiceContext) AddDBMetric(durationMs int64, isWrite bool) {
	if c == nil || c.Metrics == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Metrics.DBTotalQueries++
	if isWrite {
		c.Metrics.DBWriteMs += durationMs
	} else {
		c.Metrics.DBReadMs += durationMs
	}
}

func (c *ServiceContext) SetResult(key string, result StepResult) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Results == nil {
		c.Results = make(map[string]any)
	}
	c.Results[key] = result
}

func (c *ServiceContext) GetResult(key string) (StepResult, bool) {
	var res StepResult
	if c == nil {
		return res, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.Results[key]
	if !ok {
		return res, false
	}
	r, ok := v.(StepResult)
	if !ok {
		return res, false
	}
	return r, true
}

func (c *ServiceContext) SetInputValue(key string, value any) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Input == nil {
		c.Input = make(map[string]any)
	}
	c.Input[key] = value
}

func (c *ServiceContext) GetInputValue(key string) (any, bool) {
	if c == nil {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.Input[key]
	return v, ok
}

// GetAll returns a copy of all results in a simple map
func (c *ServiceContext) GetAll() map[string]any {
	if c == nil {
		return make(map[string]any)
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	result := make(map[string]any)
	for k, v := range c.Results {
		// If the result is a StepResult struct, we might want to extract the Data
		// or just return the StepResult. The user example showed "score": 85
		// which implies flat access.
		// However, Results stores `StepResult` or `any`?
		// SetResult stores `StepResult`.
		// Let's assume we want to return the Data part or flatten it.
		// But the user said: {"score": 85, "approved": true, ...}
		// If StepResult has Data map, maybe we should merge them?
		// Or maybe Results stores direct values?
		// Looking at SetResult: c.Results[key] = result (StepResult)
		// So Results contains StepResult objects.
		// If the user wants {"score": 85}, and "score" is a key in Results?
		// No, keys in Results are usually Step names or IDs.
		// If a step produces multiple outputs, they are in StepResult.Data.
		// So GetAll() should probably aggregate all Data from all StepResults?
		// Or maybe just return the map of StepResults?
		// The user example: `resultados := resultadoContexto.GetAll() // {"score": 85, "approved": true}`
		// implies that the context accumulates *business variables*.
		// If `Results` only stores `StepResult` keyed by step name, then we need to flatten `StepResult.Data`.

		if sr, ok := v.(StepResult); ok {
			if sr.Data != nil {
				for dataKey, dataVal := range sr.Data {
					result[dataKey] = dataVal
				}
			}
		} else {
			// Fallback if it's not a StepResult
			result[k] = v
		}
	}
	return result
}

func (c *ServiceContext) SnapshotInput() map[string]any {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Input == nil {
		return nil
	}
	out := make(map[string]any, len(c.Input))
	for k, v := range c.Input {
		out[k] = v
	}
	return out
}

func (c *ServiceContext) InitService(service Service, path string, config map[string]any) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.CurrentStepConfig = config
	service.Init(c, path)
	// Limpiamos la configuración para evitar efectos secundarios,
	// asumiendo que el servicio ya leyó lo que necesitaba en Init.
	c.CurrentStepConfig = nil
}

type Service interface {
	Init(ctx *ServiceContext, servicePath string)
	Execute() error
}
