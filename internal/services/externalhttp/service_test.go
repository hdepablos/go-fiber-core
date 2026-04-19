package externalhttp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	resty "github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testNotifier struct {
	events []Event
}

func (n *testNotifier) Notify(_ context.Context, event Event) error {
	n.events = append(n.events, event)
	return nil
}

func TestClientDo_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	notifier := &testNotifier{}
	client := NewClient(resty.New().SetBaseURL(server.URL), notifier)

	resp, err := client.Do(context.Background(), Request{
		Source:   "test_api",
		Method:   http.MethodPost,
		Endpoint: "/resource",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Empty(t, notifier.events)
}

func TestClientDo_TooManyRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"too many requests"}`))
	}))
	defer server.Close()

	notifier := &testNotifier{}
	client := NewClient(resty.New().SetBaseURL(server.URL), notifier)

	resp, err := client.Do(context.Background(), Request{
		Source:   "test_api",
		Method:   http.MethodGet,
		Endpoint: "/resource",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode())
	require.Len(t, notifier.events, 1)
	assert.Equal(t, "external_http_429", notifier.events[0].Type)
	assert.Equal(t, "30", notifier.events[0].RetryAfter)
}

func TestClientDo_ContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := &testNotifier{}
	client := NewClient(resty.New().SetBaseURL(server.URL).SetTimeout(2*time.Second), notifier)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	resp, err := client.Do(ctx, Request{
		Source:   "test_api",
		Method:   http.MethodGet,
		Endpoint: "/resource",
	})

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
	require.Len(t, notifier.events, 1)
	assert.Equal(t, "external_dependency_error", notifier.events[0].Type)
}

func TestClientDo_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(80 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := &testNotifier{}
	client := newClient(resty.New().SetBaseURL(server.URL).SetTimeout(20*time.Millisecond), notifier, 20*time.Millisecond)

	resp, err := client.Do(context.Background(), Request{
		Source:   "test_api",
		Method:   http.MethodGet,
		Endpoint: "/resource",
	})

	require.Error(t, err)
	assert.Nil(t, resp)
	require.Len(t, notifier.events, 1)
	assert.Equal(t, "external_dependency_timeout", notifier.events[0].Type)
}
