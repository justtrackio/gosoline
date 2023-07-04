//go:build integration
// +build integration

package apiserver

import (
	"context"
	netHttp "net/http"
	"testing"

	httpHeaders "github.com/go-http-utils/headers"
	"github.com/justtrackio/gosoline/pkg/apiserver"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/http"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type panicHandler struct{}

func (h panicHandler) Handle(context.Context, *apiserver.Request) (*apiserver.Response, error) {
	panic("test err")
}

type PanicTestSuite struct {
	suite.Suite
}

func (s *PanicTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithConfigFile("./config.dist.eofBind.yml"),
		suite.WithSharedEnvironment(),
		suite.WithoutAutoDetectedComponents("localstack"),
	}
}

func (s *PanicTestSuite) SetupApiDefinitions() apiserver.Definer {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (*apiserver.Definitions, error) {
		d := &apiserver.Definitions{}
		d.POST("/panic", apiserver.CreateHandler(&panicHandler{}))

		return d, nil
	}
}

func (s *PanicTestSuite) TestPanic() *suite.ApiServerTestCase {
	return &suite.ApiServerTestCase{
		Method: netHttp.MethodPost,
		Url:    "/panic",
		Headers: map[string]string{
			httpHeaders.ContentType: http.MimeTypeApplicationJson,
		},
		Body:               []byte{},
		ExpectedStatusCode: netHttp.StatusInternalServerError,
	}
}

func TestPanicTestSuite(t *testing.T) {
	suite.Run(t, &PanicTestSuite{})
}
