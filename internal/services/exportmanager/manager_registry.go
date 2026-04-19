package exportmanager

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

type ManagerResolver func(ctx context.Context) (Manager, error)

type Registry interface {
	Register(resolver ManagerResolver, executionKeys ...string)
	Resolve(ctx context.Context, executionKey string) (Manager, error)
}

type managerRegistry struct {
	mu        sync.RWMutex
	resolvers map[string]ManagerResolver
}

func NewManagerRegistry() Registry {
	return &managerRegistry{
		resolvers: make(map[string]ManagerResolver),
	}
}

func (r *managerRegistry) Register(resolver ManagerResolver, executionKeys ...string) {
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
		return nil, fmt.Errorf("export manager no registrado para execution key %q", executionKey)
	}
	return resolver(ctx)
}

var defaultManagerRegistry = NewManagerRegistry()

func RegisterManagedExportManager(resolver ManagerResolver, executionKeys ...string) {
	defaultManagerRegistry.Register(resolver, executionKeys...)
}

func ResolveManagedExportManager(ctx context.Context, executionKey string) (Manager, error) {
	return defaultManagerRegistry.Resolve(ctx, executionKey)
}
