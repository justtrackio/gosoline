// +build integration

package apiserver

import (
	"bytes"
	"compress/gzip"
	"context"
	"github.com/applike/gosoline/pkg/apiserver"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/http"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/test/suite"
	"github.com/go-resty/resty/v2"
	netHttp "net/http"
	"testing"
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

func (s *CompressedTestSuite) SetupApiDefinitions() apiserver.Definer {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (*apiserver.Definitions, error) {
		d := &apiserver.Definitions{}
		d.POST("/echo", apiserver.CreateJsonHandler(&compressedHandler{}))
		d.POST("/uncompressed", apiserver.CreateJsonHandler(&compressedHandler{}))
		d.POST("/this-path-uses-no-compression-to-echo", apiserver.CreateJsonHandler(&compressedHandler{}))

		return d, nil
	}
}

func (s *CompressedTestSuite) TestCompressed() []*suite.ApiServerTestCase {
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

	var result []*suite.ApiServerTestCase

	for i, route := range []string{"/echo", "/uncompressed", "/this-path-uses-no-compression-to-echo"} {
		i := i

		result = append(result, &suite.ApiServerTestCase{
			Method: netHttp.MethodPost,
			Url:    route,
			Headers: map[string]string{
				http.HdrContentType:     http.ContentTypeApplicationJson,
				http.HdrContentEncoding: http.ContentEncodingGzip,
				http.HdrAcceptEncoding:  http.ContentEncodingGzip,
			},
			Body:               buffer.Bytes(), // all routes should accept compressed requests
			ExpectedStatusCode: netHttp.StatusOK,
			ValidateResponse: func(res *resty.Response) error {
				// only first route should be compressed
				if i == 0 {
					s.Equal([]string{http.ContentEncodingGzip}, res.Header()[http.HdrContentEncoding])
				} else {
					s.Equal([]string(nil), res.Header()[http.HdrContentEncoding])
				}

				s.Equal([]string{apiserver.ContentTypeJson}, res.Header()[http.HdrContentType])
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

func (h compressedHandler) Handle(_ context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	body := request.Body.(*map[string]interface{})

	return apiserver.NewJsonResponse(*body), nil
}

func TestCompressedTestSuite(t *testing.T) {
	suite.Run(t, &CompressedTestSuite{})
}
