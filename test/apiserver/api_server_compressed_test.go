// +build integration

package apiserver

import (
	"bytes"
	"compress/gzip"
	"context"
	"github.com/applike/gosoline/pkg/apiserver"
	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/http"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

type handler struct{}

type testModule struct {
	kernel.EssentialModule

	t      *testing.T
	client http.Client
}

func TestApiServerCompression(t *testing.T) {
	args := os.Args
	os.Args = args[:1]
	defer func() {
		os.Args = args
	}()

	app := application.Default()
	app.Add("api", apiserver.New(defineRoutes))
	app.Add("test", newTestModule(t))
	app.Run()
}

func defineRoutes(_ context.Context, _ cfg.Config, _ mon.Logger) (*apiserver.Definitions, error) {
	d := &apiserver.Definitions{}
	d.POST("/echo", apiserver.CreateJsonHandler(&handler{}))
	d.POST("/uncompressed", apiserver.CreateJsonHandler(&handler{}))
	d.POST("/this-path-uses-no-compression-to-echo", apiserver.CreateJsonHandler(&handler{}))

	return d, nil
}

func (h handler) Handle(_ context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	body := request.Body.(*map[string]interface{})

	return apiserver.NewJsonResponse(*body), nil
}

func (h handler) GetInput() interface{} {
	return &map[string]interface{}{}
}

func newTestModule(t *testing.T) kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger mon.Logger) (kernel.Module, error) {
		client := http.NewHttpClient(config, logger)

		return &testModule{
			t:      t,
			client: client,
		}, nil
	}
}

func (t testModule) Run(ctx context.Context) error {
	// language=JSON
	bodyString := `{ "id": 42, "name": "nice json request", "content": "this is a long string. this is a long string. this is a long string. this is a long string. this is a long string. this is a long string. " }`
	// language=JSON
	expectedBody := `{"content":"this is a long string. this is a long string. this is a long string. this is a long string. this is a long string. this is a long string. ","id":42,"name":"nice json request"}`

	buffer := bytes.NewBuffer([]byte{})
	writer := gzip.NewWriter(buffer)
	_, err := writer.Write([]byte(bodyString))
	assert.NoError(t.t, err)
	err = writer.Close()
	assert.NoError(t.t, err)

	for i, route := range []string{"echo", "uncompressed", "this-path-uses-no-compression-to-echo"} {
		req := http.NewRequest(t.client).
			WithUrl("http://localhost:25399/"+route).
			WithBody(buffer.Bytes()). // all routes should accept compressed requests
			WithHeader(http.HdrContentType, http.ContentTypeApplicationJson).
			WithHeader(http.HdrContentEncoding, http.ContentEncodingGzip).
			WithHeader(http.HdrAcceptEncoding, http.ContentEncodingGzip)

		res, err := t.client.Post(ctx, req)
		assert.NoError(t.t, err)

		// only first route should be compressed
		if i == 0 {
			assert.Equal(t.t, http.ContentEncodingGzip, res.Header.Get(http.HdrContentEncoding))
		} else {
			assert.Equal(t.t, "", res.Header.Get(http.HdrContentEncoding))
		}
		assert.Equal(t.t, apiserver.ContentTypeJson, res.Header.Get(http.HdrContentType))
		assert.Equal(t.t, expectedBody, string(res.Body))
	}

	return nil
}
