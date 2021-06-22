package http_test

import (
	"context"
	"errors"
	"fmt"
	cfgMocks "github.com/applike/gosoline/pkg/cfg/mocks"
	"github.com/applike/gosoline/pkg/http"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	netHttp "net/http"
	"net/http/httptest"
	"testing"
	"time"
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

func (m *myContext) Value(key interface{}) interface{} {
	return nil
}

func runTestServer(t *testing.T, method string, status int, delay time.Duration, test func(host string)) {
	testServer := httptest.NewServer(netHttp.HandlerFunc(func(res netHttp.ResponseWriter, req *netHttp.Request) {
		assert.Equal(t, method, req.Method)

		time.Sleep(delay)

		res.WriteHeader(status)
	}))
	defer func() { testServer.Close() }()

	test(testServer.Listener.Addr().String())
}

func getConfig(retries int, timeout int) *cfgMocks.Config {
	config := new(cfgMocks.Config)
	config.On("UnmarshalKey", mock.AnythingOfType("string"), mock.AnythingOfType("*http.Settings"))
	config.On("GetInt", "http_client_retry_count").Return(retries)
	config.On("GetDuration", "http_client_request_timeout").Return(time.Duration(timeout) * time.Second)

	return config
}

func TestClient_Delete(t *testing.T) {
	config := getConfig(1, 1)
	logger := logMocks.NewLoggerMockedAll()

	runTestServer(t, "DELETE", 200, 0, func(host string) {
		client := http.NewHttpClient(config, logger)
		request := client.NewRequest().
			WithUrl(fmt.Sprintf("http://%s", host))
		response, err := client.Delete(context.TODO(), request)

		assert.NoError(t, err)
		assert.Equal(t, 200, response.StatusCode)
	})

	config.AssertExpectations(t)
}

func TestClient_Get(t *testing.T) {
	config := getConfig(1, 1)
	logger := logMocks.NewLoggerMockedAll()

	runTestServer(t, "GET", 200, 0, func(host string) {
		client := http.NewHttpClient(config, logger)
		request := client.NewRequest().
			WithUrl(fmt.Sprintf("http://%s", host))
		response, err := client.Get(context.TODO(), request)

		assert.NoError(t, err)
		assert.Equal(t, 200, response.StatusCode)
	})

	config.AssertExpectations(t)
}

func TestClient_GetTimeout(t *testing.T) {
	config := getConfig(0, 1)
	logger := logMocks.NewLoggerMockedAll()

	runTestServer(t, "GET", 200, 1100*time.Millisecond, func(host string) {
		client := http.NewHttpClient(config, logger)
		request := client.NewRequest().
			WithUrl(fmt.Sprintf("http://%s", host))
		response, err := client.Get(context.TODO(), request)

		assert.Error(t, err)
		assert.Nil(t, response)
	})

	config.AssertExpectations(t)
}

func TestClient_GetCanceled(t *testing.T) {
	config := getConfig(1, 1)
	logger := logMocks.NewLoggerMockedAll()

	baseCtx := myContext(0)
	ctx, cancel := context.WithCancel(&baseCtx)

	runTestServer(t, "GET", 200, 200*time.Millisecond, func(host string) {
		client := http.NewHttpClient(config, logger)
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

	config.AssertExpectations(t)
}

func TestClient_Post(t *testing.T) {
	config := getConfig(1, 1)
	logger := logMocks.NewLoggerMockedAll()

	runTestServer(t, "POST", 200, 0, func(host string) {
		client := http.NewHttpClient(config, logger)
		request := client.NewRequest().
			WithUrl(fmt.Sprintf("http://%s", host))
		response, err := client.Post(context.TODO(), request)

		assert.NoError(t, err)
		assert.Equal(t, 200, response.StatusCode)
	})

	config.AssertExpectations(t)
}
