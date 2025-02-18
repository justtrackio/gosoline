//go:build integration

package httpserver

import (
	"bytes"
	"compress/gzip"
	"context"
	netHttp "net/http"
	"testing"

	httpHeaders "github.com/go-http-utils/headers"
	"github.com/go-resty/resty/v2"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/http"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type compressedHandler struct{}

type CompressedTestSuite struct {
	suite.Suite
}

func (s *CompressedTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithConfigFile("./config.dist.compressed.yml"),
		suite.WithSharedEnvironment(),
		suite.WithoutAutoDetectedComponents("localstack"),
	}
}

func (s *CompressedTestSuite) SetupApiDefinitions() httpserver.Definer {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (*httpserver.Definitions, error) {
		d := &httpserver.Definitions{}
		d.POST("/echo", httpserver.CreateJsonHandler(&compressedHandler{}))
		d.POST("/uncompressed", httpserver.CreateJsonHandler(&compressedHandler{}))
		d.POST("/this-path-uses-no-compression-to-echo", httpserver.CreateJsonHandler(&compressedHandler{}))

		return d, nil
	}
}

func (s *CompressedTestSuite) TestCompressed() []*suite.HttpserverTestCase {
	// language=JSON
	bodyString := `{ "id": 42, "name": "nice json request", "content": "this is a long string. this is a long string. this is a long string. this is a long string. this is a long string. this is a long string. " }`
	// language=JSON
	expectedBody := `{"content":"this is a long string. this is a long string. this is a long string. this is a long string. this is a long string. this is a long string. ","id":42,"name":"nice json request"}`

	buffer := bytes.NewBuffer([]byte{})
	writer := gzip.NewWriter(buffer)
	_, err := writer.Write([]byte(bodyString))
	s.NoError(err)
	err = writer.Close()
	s.NoError(err)

	var result []*suite.HttpserverTestCase

	for i, route := range []string{"/echo", "/uncompressed", "/this-path-uses-no-compression-to-echo"} {
		result = append(result, &suite.HttpserverTestCase{
			Method: netHttp.MethodPost,
			Url:    route,
			Headers: map[string]string{
				httpHeaders.ContentType:     http.MimeTypeApplicationJson,
				httpHeaders.ContentEncoding: http.ContentEncodingGzip,
				httpHeaders.AcceptEncoding:  http.ContentEncodingGzip,
			},
			Body:               buffer.Bytes(), // all routes should accept compressed requests
			ExpectedStatusCode: netHttp.StatusOK,
			Assert: func(res *resty.Response) error {
				// only first route should be compressed
				if i == 0 {
					s.Equal([]string{http.ContentEncodingGzip}, res.Header()[httpHeaders.ContentEncoding])
				} else {
					s.Equal([]string(nil), res.Header()[httpHeaders.ContentEncoding])
				}

				s.Equal([]string{httpserver.ContentTypeJson}, res.Header()[httpHeaders.ContentType])
				s.Equal(expectedBody, string(res.Body()))

				return nil
			},
		})
	}

	return result
}

func (h compressedHandler) GetInput() interface{} {
	return &map[string]interface{}{}
}

func (h compressedHandler) Handle(_ context.Context, request *httpserver.Request) (*httpserver.Response, error) {
	body := request.Body.(*map[string]interface{})

	return httpserver.NewJsonResponse(*body), nil
}

func TestCompressedTestSuite(t *testing.T) {
	suite.Run(t, &CompressedTestSuite{})
}
