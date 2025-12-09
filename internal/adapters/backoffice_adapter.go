package adapters

import (
	"context"

	resty "github.com/go-resty/resty/v2" // ✅ Esta línea soluciona el error

	"go-fiber-core/internal/dtos"
)

// BackofficeAdapter es una mejor denominación, ya que su función es adaptar.
type BackofficeAdapter struct {
	client *resty.Client
}

// Constructor: Ahora recibe sus dependencias (el cliente HTTP).
// No lee archivos ni flags. Es una función pura y simple.
func NewBackofficeAdapter(client *resty.Client) *BackofficeAdapter {
	return &BackofficeAdapter{
		client: client,
	}
}

// PostReversal: Ahora recibe un context.Context como primer argumento.
// Esta es una práctica estándar para operaciones que pueden ser canceladas
// o tener un timeout, como las llamadas de red.
func (a *BackofficeAdapter) PostReversal(ctx context.Context, backofficeReversal dtos.Config) (*resty.Response, error) {
	// La URL base ya está configurada en el cliente, solo añadimos el endpoint.
	const endpoint = "/collections/collect-data"

	// La llamada a Resty ahora incluye el contexto.
	resp, err := a.client.R().
		SetContext(ctx).
		SetBody(backofficeReversal).
		SetHeader("Content-Type", "application/json").
		Post(endpoint)

	// El adaptador ya no loguea. Su única responsabilidad es ejecutar
	// la petición y devolver el resultado o el error. Quien lo llama
	// (el comando) se encargará de loguear.
	if err != nil {
		return nil, err
	}

	return resp, nil
}
