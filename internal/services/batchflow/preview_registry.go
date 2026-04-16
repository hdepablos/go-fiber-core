package batchflow

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

type PreviewProviderResolver func(ctx context.Context) (PreviewProvider, error)

type PreviewRegistry interface {
	Register(processTypeName string, resolver PreviewProviderResolver, executionKeys ...string)
	Resolve(ctx context.Context, processTypeName string, executionKeys []string) (PreviewProvider, error)
}

type previewRegistry struct {
	mu            sync.RWMutex
	nameResolvers map[string]PreviewProviderResolver
	keyResolvers  map[string]PreviewProviderResolver
}

func NewPreviewRegistry() PreviewRegistry {
	return &previewRegistry{
		nameResolvers: make(map[string]PreviewProviderResolver),
		keyResolvers:  make(map[string]PreviewProviderResolver),
	}
}

func (r *previewRegistry) Register(processTypeName string, resolver PreviewProviderResolver, executionKeys ...string) {
	if resolver == nil {
		return
	}
	key := normalizeProcessTypeName(processTypeName)
	r.mu.Lock()
	defer r.mu.Unlock()
	if key != "" {
		r.nameResolvers[key] = resolver
	}
	for _, executionKey := range executionKeys {
		executionKey = strings.TrimSpace(executionKey)
		if executionKey == "" {
			continue
		}
		r.keyResolvers[executionKey] = resolver
	}
}

func (r *previewRegistry) Resolve(ctx context.Context, processTypeName string, executionKeys []string) (PreviewProvider, error) {
	r.mu.RLock()
	for _, executionKey := range executionKeys {
		if resolver, ok := r.keyResolvers[strings.TrimSpace(executionKey)]; ok {
			r.mu.RUnlock()
			return resolver(ctx)
		}
	}
	key := normalizeProcessTypeName(processTypeName)
	resolver, ok := r.nameResolvers[key]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("preview no registrado para process_type %q", processTypeName)
	}
	return resolver(ctx)
}

var defaultPreviewRegistry = NewPreviewRegistry()

func RegisterPreviewProvider(processTypeName string, resolver PreviewProviderResolver, executionKeys ...string) {
	defaultPreviewRegistry.Register(processTypeName, resolver, executionKeys...)
}

func normalizeProcessTypeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
