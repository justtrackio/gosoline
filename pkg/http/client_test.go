package http_test

import (
	"context"
	"errors"
	"fmt"
	netHttp "net/http"
	"net/http/httptest"
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
