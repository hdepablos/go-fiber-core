package adapters

import (
	"context"
	"net/http"

	resty "github.com/go-resty/resty/v2" // ✅ Esta línea soluciona el error

	"go-fiber-core/internal/dtos"
	"go-fiber-core/internal/dtos/config"
	"go-fiber-core/internal/services/externalhttp"
)

// BackofficeAdapter es una mejor denominación, ya que su función es adaptar.
type BackofficeAdapter struct {
	httpClient externalhttp.Client
}

func NewBackofficeAdapter(cfg config.ApiConfig) *BackofficeAdapter {
	return NewBackofficeAdapterWithService(cfg, externalhttp.NewClientFromAPIConfig(cfg, nil))
}

func NewBackofficeAdapterWithService(_ config.ApiConfig, httpClient externalhttp.Client) *BackofficeAdapter {
	return &BackofficeAdapter{
		httpClient: httpClient,
	}
}

// PostReversal: Ahora recibe un context.Context como primer argumento.
// Esta es una práctica estándar para operaciones que pueden ser canceladas
// o tener un timeout, como las llamadas de red.
func (a *BackofficeAdapter) PostReversal(ctx context.Context, backofficeReversal dtos.Config) (*resty.Response, error) {
	// La URL base ya está configurada en el cliente, solo añadimos el endpoint.
	const endpoint = "/collections/collect-data"
	return a.httpClient.Do(ctx, externalhttp.Request{
		Source:   "backoffice",
		Method:   http.MethodPost,
		Endpoint: endpoint,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: backofficeReversal,
	})
}
