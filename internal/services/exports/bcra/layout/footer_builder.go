package layout

import (
	"context"

	"go-fiber-core/internal/services/exportmanager"
)

type footerBuilder struct{}

// NewFooterBuilder crea la pieza que construye el cierre del archivo.
func NewFooterBuilder() exportmanager.FooterBuilder {
	return &footerBuilder{}
}

// BuildFooter arma las lineas finales del archivo usando el Summary y el runtime del export.
func (b *footerBuilder) BuildFooter(ctx context.Context, execCtx exportmanager.ExecutionContext) ([]string, error) {
	_ = execCtx

	line, err := buildFooterLine(ctx)
	if err != nil {
		return nil, err
	}
	return []string{line}, nil
}
