package contracts

import (
	"context"
	"sync"
)

type StepResult struct {
	Status  string         `json:"status"`
	Message string         `json:"message,omitempty"`
	Input   map[string]any `json:"input,omitempty"`
	Data    map[string]any `json:"data,omitempty"`
}

type ServiceContext struct {
	Ctx               context.Context `json:"-"`
	mu                sync.Mutex
	Input             map[string]any  `json:"input,omitempty"`
	Results           map[string]any `json:"results"`
	CurrentStepConfig map[string]any `json:"-"`
}

func NewServiceContext(age, salary int) *ServiceContext {
	return NewServiceContextWithCtx(context.Background(), age, salary)
}

func NewServiceContextWithCtx(ctx context.Context, age, salary int) *ServiceContext {
	if ctx == nil {
		ctx = context.Background()
	}
	return &ServiceContext{
		Ctx:    ctx,
		Input:  map[string]any{"age": age, "salary": salary},
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
	if c.Input == nil {
		return nil, false
	}
	v, ok := c.Input[key]
	return v, ok
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

type Service interface {
	Init(ctx *ServiceContext, servicePath string)
	Execute() error
}
