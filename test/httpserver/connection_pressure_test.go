//go:build integration

package httpserver

import (
	"bufio"
	"context"
	"net"
	netHttp "net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type ConnectionPressureTestSuite struct {
	suite.Suite
}

func (s *ConnectionPressureTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithConfigFile("./config.dist.connectionPressure.yml"),
		suite.WithSharedEnvironment(),
		suite.WithoutAutoDetectedComponents("localstack"),
	}
}

func (s *ConnectionPressureTestSuite) SetupApiDefinitions() httpserver.Definer {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (*httpserver.Definitions, error) {
		d := &httpserver.Definitions{}
		d.GET("/ok", func(c *gin.Context) {
			c.Status(netHttp.StatusNoContent)
		})

		return d, nil
	}
}

func (s *ConnectionPressureTestSuite) TestClosesIdleConnectionUnderConnectionPressure(_ suite.AppUnderTest, client *resty.Client) error {
	_, port, err := net.SplitHostPort(client.BaseURL[len("http://"):])
	s.Require().NoError(err)

	connA, err := net.Dial("tcp", net.JoinHostPort("127.0.0.1", port))
	s.Require().NoError(err)
	defer func() {
		s.Require().NoError(connA.Close())
	}()

	_, err = connA.Write([]byte("GET /ok HTTP/1.1\r\nHost: localhost\r\n\r\n"))
	s.Require().NoError(err)

	s.readStatus(connA, netHttp.StatusNoContent)

	connB, err := net.Dial("tcp", net.JoinHostPort("127.0.0.1", port))
	s.Require().NoError(err)
	defer func() {
		s.Require().NoError(connB.Close())
	}()

	_, err = connB.Write([]byte("GET /ok HTTP/1.1\r\nHost: localhost\r\n\r\n"))
	s.Require().NoError(err)

	s.readStatus(connB, netHttp.StatusNoContent)
	s.Require().NoError(connA.SetReadDeadline(time.Now().Add(time.Second)))

	_, err = netHttp.ReadResponse(bufio.NewReader(connA), nil)
	s.Require().Error(err)

	return nil
}

func (s *ConnectionPressureTestSuite) readStatus(conn net.Conn, expectedStatus int) {
	response, err := netHttp.ReadResponse(bufio.NewReader(conn), nil)
	s.Require().NoError(err)
	defer func() {
		s.Require().NoError(response.Body.Close())
	}()

	s.Equal(expectedStatus, response.StatusCode)
}

func TestConnectionPressureTestSuite(t *testing.T) {
	suite.Run(t, &ConnectionPressureTestSuite{})
}
