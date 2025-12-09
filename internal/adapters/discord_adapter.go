package adapters

import (
	"context"
	"time"

	resty "github.com/go-resty/resty/v2"

	"go-fiber-core/internal/dtos"
	"go-fiber-core/internal/dtos/config"
)

type DiscordAdapter struct {
	client *resty.Client
	// CAMBIO: El tipo de la configuración es 'ApiConfig', no 'ApiDiscord'.
	apiConfig config.ApiConfig
}

// NewDiscordAdapter recibe la configuración genérica 'ApiConfig'.
func NewDiscordAdapter(cfg config.ApiConfig) *DiscordAdapter {
	client := resty.New().
		SetTimeout(10*time.Second).
		SetHeader("Content-Type", "application/json")

	return &DiscordAdapter{
		client:    client,
		apiConfig: cfg,
	}
}

// El método Send no necesita cambios.
func (a *DiscordAdapter) Send(ctx context.Context, notification dtos.NotificationDiscord) (*resty.Response, error) {
	resp, err := a.client.R().
		SetContext(ctx).
		SetBody(notification).
		Post(a.apiConfig.Url)

	if err != nil {
		return nil, err
	}

	return resp, nil
}
