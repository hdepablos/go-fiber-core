package layout

import (
	"context"

	"go-fiber-core/internal/services/exportmanager"
)

type headerBuilder struct{}

// NewHeaderBuilder crea la pieza que construye el encabezado del archivo final.
func NewHeaderBuilder() exportmanager.HeaderBuilder {
	return &headerBuilder{}
}

// BuildHeader arma las primeras lineas del archivo usando el contexto global del export.
func (b *headerBuilder) BuildHeader(ctx context.Context, execCtx exportmanager.ExecutionContext) ([]string, error) {
	_ = execCtx

	line, err := buildHeaderLine(ctx)
	if err != nil {
		return nil, err
	}
	return []string{line}, nil
}
