package runtime

import (
	"context"
	"fmt"

	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/services/exportmanager"
	"go-fiber-core/internal/services/runtimectx"
)

type Provider interface {
	Manager() exportmanager.Manager
	Connect() *connect.ConnectDTO
}

const providerContextKey = "bcra.provider"

// WithProvider adjunta el provider del export al contexto actual.
func WithProvider(ctx context.Context, prov Provider) context.Context {
	return runtimectx.WithNamedValue(ctx, providerContextKey, prov)
}

// ProviderFromContext recupera el provider previamente inyectado para este export.
func ProviderFromContext(ctx context.Context) (Provider, error) {
	if prov, ok := runtimectx.NamedValue[Provider](ctx, providerContextKey); ok && prov != nil {
		return prov, nil
	}
	return nil, fmt.Errorf("provider no disponible en contexto")
}
