package layout

import (
	"context"
	"encoding/json"

	"go-fiber-core/internal/services/exportmanager"
)

type bodyBuilder struct{}

// NewBodyBuilder crea la pieza que transforma cada item en lineas del cuerpo del archivo.
func NewBodyBuilder() exportmanager.BodyBuilder {
	return &bodyBuilder{}
}

// BuildBodyLines conserva el contrato del framework y delega la logica real del item en renderItem.
func (b *bodyBuilder) BuildBodyLines(ctx context.Context, execCtx exportmanager.ExecutionContext, item json.RawMessage) ([]string, error) {
	return b.renderItem(ctx, execCtx, item)
}

// renderItem es el punto de extension estandar del export.
// Aqui el developer transforma un registro del batch en una o varias lineas del archivo.
func (b *bodyBuilder) renderItem(ctx context.Context, execCtx exportmanager.ExecutionContext, item json.RawMessage) ([]string, error) {
	_ = execCtx

	line, err := buildBodyLine(ctx, item)
	if err != nil {
		return nil, err
	}
	return []string{line}, nil
}
