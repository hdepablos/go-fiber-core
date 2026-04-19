package adapters

import (
	"context"
	"net/http"

	resty "github.com/go-resty/resty/v2"

	"go-fiber-core/internal/dtos"
	"go-fiber-core/internal/dtos/config"
	"go-fiber-core/internal/services/externalhttp"
)

type DiscordAdapter struct {
	httpClient externalhttp.Client
	// CAMBIO: El tipo de la configuración es 'ApiConfig', no 'ApiDiscord'.
	apiConfig config.ApiConfig
}

// NewDiscordAdapter recibe la configuración genérica 'ApiConfig'.
func NewDiscordAdapter(cfg config.ApiConfig) *DiscordAdapter {
	return NewDiscordAdapterWithService(cfg, externalhttp.NewClientFromAPIConfig(cfg, nil))
}

func NewDiscordAdapterWithService(cfg config.ApiConfig, httpClient externalhttp.Client) *DiscordAdapter {
	return &DiscordAdapter{
		httpClient: httpClient,
		apiConfig:  cfg,
	}
}

func (a *DiscordAdapter) Send(ctx context.Context, notification dtos.NotificationDiscord) (*resty.Response, error) {
	return a.httpClient.Do(ctx, externalhttp.Request{
		Source:   "discord",
		Method:   http.MethodPost,
		Endpoint: a.apiConfig.Url,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: notification,
	})
}
