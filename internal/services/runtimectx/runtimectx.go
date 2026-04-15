package runtimectx

import (
	"context"

	"go-fiber-core/internal/services/dispatcher"
)

type contextKey string

const (
	dispatcherKey contextKey = "runtime.dispatcher"
	valuePrefix   string     = "runtime.value."
)

func WithDispatcher(ctx context.Context, d dispatcher.Dispatcher) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, dispatcherKey, d)
}

func Dispatcher(ctx context.Context) (dispatcher.Dispatcher, bool) {
	if ctx == nil {
		return nil, false
	}
	d, ok := ctx.Value(dispatcherKey).(dispatcher.Dispatcher)
	return d, ok && d != nil
}

func WithNamedValue(ctx context.Context, name string, value any) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, contextKey(valuePrefix+name), value)
}

func NamedValue[T any](ctx context.Context, name string) (T, bool) {
	var zero T
	if ctx == nil {
		return zero, false
	}
	value, ok := ctx.Value(contextKey(valuePrefix + name)).(T)
	if !ok {
		return zero, false
	}
	return value, true
}
