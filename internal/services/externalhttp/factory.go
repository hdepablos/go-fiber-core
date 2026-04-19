package externalhttp

import (
	"strings"
	"time"

	"go-fiber-core/internal/dtos/config"

	resty "github.com/go-resty/resty/v2"
)

const defaultTimeout = 10 * time.Second

func NewRestyClientFromAPIConfig(cfg config.ApiConfig) *resty.Client {
	timeout := defaultTimeout
	if cfg.TimeoutSeconds > 0 {
		timeout = time.Duration(cfg.TimeoutSeconds) * time.Second
	}

	client := resty.New().
		SetTimeout(timeout).
		SetHeader("Content-Type", "application/json")

	if strings.TrimSpace(cfg.Url) != "" {
		client.SetBaseURL(strings.TrimSpace(cfg.Url))
	}
	if strings.TrimSpace(cfg.Token) != "" {
		client.SetAuthToken(strings.TrimSpace(cfg.Token))
	}
	for key, value := range cfg.Headers {
		client.SetHeader(key, value)
	}

	return client
}

func NewClientFromAPIConfig(cfg config.ApiConfig, notifier Notifier) Client {
	timeout := defaultTimeout
	if cfg.TimeoutSeconds > 0 {
		timeout = time.Duration(cfg.TimeoutSeconds) * time.Second
	}
	return newClient(NewRestyClientFromAPIConfig(cfg), notifier, timeout)
}
