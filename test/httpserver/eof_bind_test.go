//go:build integration
// +build integration

package httpserver

import (
	"context"
	"fmt"
	netHttp "net/http"
	"testing"
	"time"

	httpHeaders "github.com/go-http-utils/headers"
	"github.com/go-resty/resty/v2"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/http"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type eofBindHandler struct{}

type EofBindTestSuite struct {
	suite.Suite
}

func (s *EofBindTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithConfigFile("./config.dist.eofBind.yml"),
		suite.WithSharedEnvironment(),
		suite.WithoutAutoDetectedComponents("localstack"),
	}
}

func (s *EofBindTestSuite) SetupApiDefinitions() httpserver.Definer {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (*httpserver.Definitions, error) {
		d := &httpserver.Definitions{}
		d.POST("/bind", httpserver.CreateJsonHandler(&eofBindHandler{}))

		return d, nil
	}
}

type bindHandler struct {
	suite.Suite
}

func (b bindHandler) ChannelLevel(string) (level *int, err error) {
	return nil, nil
}

func (b bindHandler) Level() int {
	return log.LevelPriority(log.LevelWarn)
}

func (b bindHandler) Log(_ time.Time, _ int, msg string, args []any, _ error, _ log.Data) error {
	formattedMsg := fmt.Sprintf(msg, args...)

	b.NotEqual("POST /bind HTTP/1.1 - bind error - EOF", formattedMsg)
	b.NotEqual("POST /bind HTTP/1.1 - bind error - unexpected EOF", formattedMsg)
	b.Contains([]string{
		"POST /bind HTTP/1.1 - network error - client has gone away - EOF",
		"POST /bind HTTP/1.1 - network error - client has gone away - unexpected EOF",
	}, formattedMsg)

	return nil
}

func (s *EofBindTestSuite) TestEofBind() *suite.HttpserverTestCase {
	// language=JSON
	expectedBody := `{"err":"EOF"}`

	err := s.Env().Logger().Option(log.WithHandlers(bindHandler{
		Suite: s.Suite,
	}))
	s.NoError(err)

	return &suite.HttpserverTestCase{
		Method: netHttp.MethodPost,
		Url:    "/bind",
		Headers: map[string]string{
			httpHeaders.ContentType: http.MimeTypeApplicationJson,
		},
		Body:               []byte{},
		ExpectedStatusCode: netHttp.StatusBadRequest,
		Assert: func(res *resty.Response) error {
			s.Equal(expectedBody, string(res.Body()))

			return nil
		},
	}
}

func (s *EofBindTestSuite) TestUnexpectedEofBind() *suite.HttpserverTestCase {
	// language=JSON
	expectedBody := `{"err":"unexpected EOF"}`

	err := s.Env().Logger().Option(log.WithHandlers(bindHandler{
		Suite: s.Suite,
	}))
	s.NoError(err)

	return &suite.HttpserverTestCase{
		Method: netHttp.MethodPost,
		Url:    "/bind",
		Headers: map[string]string{
			httpHeaders.ContentType: http.MimeTypeApplicationJson,
		},
		Body:               []byte{'{'},
		ExpectedStatusCode: netHttp.StatusBadRequest,
		Assert: func(res *resty.Response) error {
			s.Equal(expectedBody, string(res.Body()))

			return nil
		},
	}
}

func (h eofBindHandler) GetInput() any {
	return &map[string]any{}
}

func (h eofBindHandler) Handle(_ context.Context, request *httpserver.Request) (*httpserver.Response, error) {
	body := request.Body.(*map[string]any)

	return httpserver.NewJsonResponse(*body), nil
}

func TestEofBindTestSuite(t *testing.T) {
	suite.Run(t, &EofBindTestSuite{})
}
