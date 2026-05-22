//go:build integration

package httpserver

import (
	"context"
	"fmt"
	netHttp "net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/conc"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type ConcurrencyTestSuite struct {
	suite.Suite

	enteredHandler conc.SignalOnce
	releaseHandler conc.SignalOnce
}

func (s *ConcurrencyTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithConfigFile("./config.dist.concurrency.yml"),
		suite.WithSharedEnvironment(),
		suite.WithoutAutoDetectedComponents("localstack"),
	}
}

func (s *ConcurrencyTestSuite) SetupTest() error {
	s.enteredHandler = conc.NewSignalOnce()
	s.releaseHandler = conc.NewSignalOnce()

	return nil
}

func (s *ConcurrencyTestSuite) SetupApiDefinitions() httpserver.Definer {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (*httpserver.Definitions, error) {
		d := &httpserver.Definitions{}
		d.GET("/block", func(c *gin.Context) {
			s.enteredHandler.Signal()

			<-s.releaseHandler.Channel()
			c.Status(netHttp.StatusNoContent)
		})
		d.GET("/ok", func(c *gin.Context) {
			c.Status(netHttp.StatusNoContent)
		})

		return d, nil
	}
}

func (s *ConcurrencyTestSuite) TestRejectsRequestWhenConcurrentRequestLimitIsReached(_ suite.AppUnderTest, client *resty.Client) error {
	firstDone := make(chan error, 1)
	go func() {
		res, err := client.R().Get("/block")
		if err != nil {
			firstDone <- err

			return
		}

		if res.StatusCode() != netHttp.StatusNoContent {
			firstDone <- fmt.Errorf("expected first request status %d, got %d", netHttp.StatusNoContent, res.StatusCode())

			return
		}

		firstDone <- nil
	}()

	s.Require().Eventually(s.enteredHandler.Signaled, time.Second, time.Millisecond)

	res, err := client.R().Get("/ok")
	s.Require().NoError(err)
	s.Equal(netHttp.StatusTooManyRequests, res.StatusCode())
	s.Equal("1", res.Header().Get("Retry-After"))
	s.JSONEq(`{"error":"server overloaded"}`, string(res.Body()))

	s.releaseHandler.Signal()
	s.NoError(<-firstDone)

	return nil
}

func TestConcurrencyTestSuite(t *testing.T) {
	suite.Run(t, &ConcurrencyTestSuite{})
}
