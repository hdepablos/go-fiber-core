package externalhttp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"go-fiber-core/internal/logger"

	resty "github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

type Client interface {
	Do(ctx context.Context, req Request) (*resty.Response, error)
}

type Notifier interface {
	Notify(ctx context.Context, event Event) error
}

type Request struct {
	Source      string
	Method      string
	Endpoint    string
	Headers     map[string]string
	QueryParams map[string]string
	Body        any
}

type Event struct {
	Type       string
	Source     string
	Method     string
	Endpoint   string
	StatusCode int
	RetryAfter string
	Err        error
}

type service struct {
	client   *resty.Client
	notifier Notifier
	timeout  time.Duration
}

type noopNotifier struct{}

func NewClient(client *resty.Client, notifier Notifier) Client {
	return newClient(client, notifier, 0)
}

func newClient(client *resty.Client, notifier Notifier, timeout time.Duration) Client {
	if notifier == nil {
		notifier = noopNotifier{}
	}
	return &service{
		client:   client,
		notifier: notifier,
		timeout:  timeout,
	}
}

func (noopNotifier) Notify(_ context.Context, _ Event) error {
	return nil
}

func (s *service) Do(ctx context.Context, req Request) (*resty.Response, error) {
	if s.client == nil {
		return nil, fmt.Errorf("resty client inválido")
	}

	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodGet
	}
	if strings.TrimSpace(req.Source) == "" {
		req.Source = "external_http"
	}
	if strings.TrimSpace(req.Endpoint) == "" {
		return nil, fmt.Errorf("endpoint requerido")
	}

	restyReq := s.client.R().SetContext(ctx)
	for key, value := range req.Headers {
		restyReq.SetHeader(key, value)
	}
	if len(req.QueryParams) > 0 {
		restyReq.SetQueryParams(req.QueryParams)
	}
	if req.Body != nil {
		restyReq.SetBody(req.Body)
	}

	var (
		resp *resty.Response
		err  error
	)

	switch method {
	case http.MethodGet:
		resp, err = restyReq.Get(req.Endpoint)
	case http.MethodPost:
		resp, err = restyReq.Post(req.Endpoint)
	case http.MethodPut:
		resp, err = restyReq.Put(req.Endpoint)
	case http.MethodPatch:
		resp, err = restyReq.Patch(req.Endpoint)
	case http.MethodDelete:
		resp, err = restyReq.Delete(req.Endpoint)
	default:
		return nil, fmt.Errorf("método HTTP no soportado: %s", method)
	}

	if err != nil {
		if isTimeoutError(err) {
			logger.LogExternalDependencyTimeout(
				req.Source,
				err,
				zap.String("component", "external_http_client"),
				zap.String("method", method),
				zap.String("endpoint", req.Endpoint),
				zap.String("base_url", s.client.BaseURL),
				zap.Duration("configured_timeout", s.timeout),
			)
			s.notify(ctx, Event{
				Type:     "external_dependency_timeout",
				Source:   req.Source,
				Method:   method,
				Endpoint: req.Endpoint,
				Err:      err,
			})
			return nil, err
		}
		logger.LogExternalDependencyError(
			req.Source,
			err,
			zap.String("component", "external_http_client"),
			zap.String("method", method),
			zap.String("endpoint", req.Endpoint),
			zap.String("base_url", s.client.BaseURL),
		)
		s.notify(ctx, Event{
			Type:     "external_dependency_error",
			Source:   req.Source,
			Method:   method,
			Endpoint: req.Endpoint,
			Err:      err,
		})
		return nil, err
	}

	if resp.StatusCode() == http.StatusTooManyRequests {
		retryAfter := resp.Header().Get("Retry-After")
		logger.LogExternalHTTPRateLimit(
			req.Source,
			resp.StatusCode(),
			zap.String("component", "external_http_client"),
			zap.String("method", method),
			zap.String("endpoint", req.Endpoint),
			zap.String("base_url", s.client.BaseURL),
			zap.String("retry_after", retryAfter),
		)
		s.notify(ctx, Event{
			Type:       "external_http_429",
			Source:     req.Source,
			Method:     method,
			Endpoint:   req.Endpoint,
			StatusCode: resp.StatusCode(),
			RetryAfter: retryAfter,
		})
		return resp, nil
	}

	if resp.StatusCode() >= http.StatusBadRequest {
		s.notify(ctx, Event{
			Type:       "external_http_non_success",
			Source:     req.Source,
			Method:     method,
			Endpoint:   req.Endpoint,
			StatusCode: resp.StatusCode(),
		})
	}

	return resp, nil
}

func (s *service) notify(ctx context.Context, event Event) {
	if s.notifier == nil {
		return
	}
	if err := s.notifier.Notify(ctx, event); err != nil {
		logger.GetLogger("external-http-client").Warn(
			"external http notifier failed",
			zap.String("source", event.Source),
			zap.String("event_type", event.Type),
			zap.Error(err),
		)
	}
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
