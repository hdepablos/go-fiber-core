package batchflow

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

type BatchManagerResolver func(ctx context.Context) (Manager, error)

type ManagerRegistry interface {
	Register(resolver BatchManagerResolver, executionKeys ...string)
	Resolve(ctx context.Context, executionKey string) (Manager, error)
}

type managerRegistry struct {
	mu        sync.RWMutex
	resolvers map[string]BatchManagerResolver
}

func NewManagerRegistry() ManagerRegistry {
	return &managerRegistry{
		resolvers: make(map[string]BatchManagerResolver),
	}
}

func (r *managerRegistry) Register(resolver BatchManagerResolver, executionKeys ...string) {
	if resolver == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, executionKey := range executionKeys {
		executionKey = strings.TrimSpace(executionKey)
		if executionKey == "" {
			continue
		}
		r.resolvers[executionKey] = resolver
	}
}

func (r *managerRegistry) Resolve(ctx context.Context, executionKey string) (Manager, error) {
	executionKey = strings.TrimSpace(executionKey)
	if executionKey == "" {
		return nil, fmt.Errorf("execution key vacia")
	}
	r.mu.RLock()
	resolver, ok := r.resolvers[executionKey]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("manager batch no registrado para execution key %q", executionKey)
	}
	return resolver(ctx)
}

var defaultManagerRegistry = NewManagerRegistry()

func RegisterManagedBatchManager(resolver BatchManagerResolver, executionKeys ...string) {
	defaultManagerRegistry.Register(resolver, executionKeys...)
}

func ResolveManagedBatchManager(ctx context.Context, executionKey string) (Manager, error) {
	return defaultManagerRegistry.Resolve(ctx, executionKey)
}
