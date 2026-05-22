package http_test

import (
	"context"
	"errors"
	"fmt"
	netHttp "net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/http"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/assert"
)

// A copy of context.emptyCtx
type myContext int

func (m *myContext) Deadline() (deadline time.Time, ok bool) {
	return
}

func (m *myContext) Done() <-chan struct{} {
	return nil
}

func (m *myContext) Err() error {
	return nil
}

func (m *myContext) Value(_ any) any {
	return nil
}

func runTestServer(t *testing.T, method string, status int, delay time.Duration, test func(host string)) {
	testServer := httptest.NewServer(netHttp.HandlerFunc(func(res netHttp.ResponseWriter, req *netHttp.Request) {
		assert.Equal(t, method, req.Method)

		time.Sleep(delay)

		res.WriteHeader(status)
	}))
	defer testServer.Close()

	test(testServer.Listener.Addr().String())
}

func getConfig(t *testing.T, retries int, timeout time.Duration) cfg.Config {
	config := cfg.New()
	err := config.Option(cfg.WithConfigMap(map[string]any{
		"http_client": map[string]any{
			"default": map[string]any{
				"request_timeout":     timeout,
				"retry_count":         retries,
				"retry_max_wait_time": "200ms",
			},
		},
	}))
	assert.NoError(t, err)

	return config
}

func getClient(t *testing.T, retries int, timeout time.Duration) http.Client {
	ctx := appctx.WithContainer(t.Context())
	config := getConfig(t, retries, timeout)

	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))

	client, err := http.ProvideHttpClient(ctx, config, logger, "default")
	assert.NoError(t, err)

	return client
}

func getRetryAfterClient(t *testing.T, retries int, timeout time.Duration, defaultWaitTime time.Duration) http.Client {
	ctx := appctx.WithContainer(t.Context())
	config := cfg.New()
	err := config.Option(cfg.WithConfigMap(map[string]any{
		"http_client": map[string]any{
			"default": map[string]any{
				"request_timeout":     timeout,
				"retry_count":         retries,
				"retry_max_wait_time": "200ms",
				"retry_wait_time":     "1ms",
				"retry_after": map[string]any{
					"default_wait_time": defaultWaitTime,
					"jitter_min":        1,
					"jitter_max":        1,
				},
			},
		},
	}))
	assert.NoError(t, err)

	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))

	client, err := http.ProvideHttpClient(ctx, config, logger, "default")
	assert.NoError(t, err)
	client.AddRetryCondition(func(response *http.Response, err error) bool {
		return err != nil || response.StatusCode == netHttp.StatusServiceUnavailable
	})

	return client
}

func TestClient_Delete(t *testing.T) {
	runTestServer(t, "DELETE", 200, 0, func(host string) {
		client := getClient(t, 1, time.Second)
		request := client.NewRequest().
			WithUrl(fmt.Sprintf("http://%s", host))
		response, err := client.Delete(t.Context(), request)

		assert.NoError(t, err)
		assert.Equal(t, 200, response.StatusCode)
	})
}

func TestClient_Get(t *testing.T) {
	runTestServer(t, "GET", 200, 0, func(host string) {
		client := getClient(t, 1, time.Second)
		request := client.NewRequest().
			WithUrl(fmt.Sprintf("http://%s", host))
		response, err := client.Get(t.Context(), request)

		assert.NoError(t, err)
		assert.Equal(t, 200, response.StatusCode)
	})
}

func TestClient_GetRetriesAfterServiceUnavailable(t *testing.T) {
	if testing.Short() {
		t.Skip("uses Retry-After second precision")
	}

	attempts := atomic.Int64{}
	testServer := httptest.NewServer(netHttp.HandlerFunc(func(res netHttp.ResponseWriter, req *netHttp.Request) {
		assert.Equal(t, netHttp.MethodGet, req.Method)

		if attempts.Add(1) == 1 {
			res.Header().Set("Retry-After", "1")
			res.WriteHeader(netHttp.StatusServiceUnavailable)

			return
		}

		res.WriteHeader(netHttp.StatusOK)
	}))
	defer testServer.Close()

	client := getRetryAfterClient(t, 1, 2*time.Second, time.Millisecond)
	request := client.NewRequest().WithUrl(testServer.URL)

	startedAt := time.Now()
	response, err := client.Get(t.Context(), request)

	assert.NoError(t, err)
	assert.Equal(t, netHttp.StatusOK, response.StatusCode)
	assert.Equal(t, int64(2), attempts.Load())
	assert.GreaterOrEqual(t, time.Since(startedAt), time.Second)
}

func TestClient_GetRetriesAfterDefaultOnMissingHeader(t *testing.T) {
	attempts := atomic.Int64{}
	testServer := httptest.NewServer(netHttp.HandlerFunc(func(res netHttp.ResponseWriter, req *netHttp.Request) {
		assert.Equal(t, netHttp.MethodGet, req.Method)

		if attempts.Add(1) == 1 {
			res.WriteHeader(netHttp.StatusServiceUnavailable)

			return
		}

		res.WriteHeader(netHttp.StatusOK)
	}))
	defer testServer.Close()

	client := getRetryAfterClient(t, 1, time.Second, 10*time.Millisecond)
	request := client.NewRequest().WithUrl(testServer.URL)

	response, err := client.Get(t.Context(), request)

	assert.NoError(t, err)
	assert.Equal(t, netHttp.StatusOK, response.StatusCode)
	assert.Equal(t, int64(2), attempts.Load())
}

func TestClient_GetFailsWhenRetryAfterExceedsRequestTimeout(t *testing.T) {
	attempts := atomic.Int64{}
	testServer := httptest.NewServer(netHttp.HandlerFunc(func(res netHttp.ResponseWriter, req *netHttp.Request) {
		assert.Equal(t, netHttp.MethodGet, req.Method)
		attempts.Add(1)
		res.Header().Set("Retry-After", "1")
		res.WriteHeader(netHttp.StatusServiceUnavailable)
	}))
	defer testServer.Close()

	client := getRetryAfterClient(t, 1, 50*time.Millisecond, time.Millisecond)
	request := client.NewRequest().WithUrl(testServer.URL)

	response, err := client.Get(t.Context(), request)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "retry wait duration")
	assert.Nil(t, response)
	assert.Equal(t, int64(1), attempts.Load())
}

func TestClient_GetTimeout(t *testing.T) {
	runTestServer(t, "GET", 200, 1100*time.Millisecond, func(host string) {
		client := getClient(t, 0, time.Second)
		request := client.NewRequest().
			WithUrl(fmt.Sprintf("http://%s", host))
		response, err := client.Get(t.Context(), request)

		assert.Error(t, err)
		assert.Nil(t, response)
	})
}

func TestClient_GetAppliesRequestTimeoutToRequestContext(t *testing.T) {
	requestDone := atomic.Bool{}
	testServer := httptest.NewServer(netHttp.HandlerFunc(func(res netHttp.ResponseWriter, req *netHttp.Request) {
		assert.Equal(t, netHttp.MethodGet, req.Method)

		<-req.Context().Done()
		requestDone.Store(true)

		res.WriteHeader(netHttp.StatusNoContent)
	}))
	defer testServer.Close()

	client := getClient(t, 0, 10*time.Millisecond)
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	request := client.NewRequest().WithUrl(testServer.URL)
	response, err := client.Get(ctx, request)

	assert.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Nil(t, response)
	assert.Eventually(t, requestDone.Load, time.Second, time.Millisecond)
}

func TestClient_GetCanceled(t *testing.T) {
	baseCtx := myContext(0)
	ctx, cancel := context.WithCancel(&baseCtx)

	runTestServer(t, "GET", 200, 200*time.Millisecond, func(host string) {
		client := getClient(t, 1, time.Second)
		request := client.NewRequest().
			WithUrl(fmt.Sprintf("http://%s", host))
		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()
		response, err := client.Get(ctx, request)

		assert.Error(t, err)
		assert.True(t, errors.Is(err, context.Canceled))
		assert.Nil(t, response)
	})
}

func TestClient_Post(t *testing.T) {
	runTestServer(t, "POST", 200, 0, func(host string) {
		client := getClient(t, 1, time.Second)
		request := client.NewRequest().
			WithUrl(fmt.Sprintf("http://%s", host))
		response, err := client.Post(t.Context(), request)

		assert.NoError(t, err)
		assert.Equal(t, 200, response.StatusCode)
	})
}
