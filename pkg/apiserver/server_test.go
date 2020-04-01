package apiserver_test

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/apiserver"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	monMocks "github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/applike/gosoline/pkg/tracing"
	tracingMocks "github.com/applike/gosoline/pkg/tracing/mocks"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

type ServerTestSuite struct {
	suite.Suite
	logger mon.Logger
	router *gin.Engine
	tracer tracing.Tracer
	server *apiserver.ApiServer
}

func TestServerTestSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}

func (s *ServerTestSuite) SetupTest() {
	s.logger = monMocks.NewLoggerMockedAll()

	gin.SetMode(gin.TestMode)
	s.router = gin.New()

	tracer := new(tracingMocks.Tracer)
	tracer.On("HttpHandler", s.router).Return(s.router)
	s.tracer = tracer

	s.server = apiserver.New(func(_ cfg.Config, _ mon.Logger, _ *apiserver.Definitions) {})
}

func (s *ServerTestSuite) TestLifecycle_Cancel() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	assert.NotPanics(s.T(), func() {
		err := s.server.BootWithInterfaces(s.logger, s.router, s.tracer, &apiserver.Settings{})
		assert.NoError(s.T(), err)

		err = s.server.Run(ctx)
		assert.NoError(s.T(), err)
	})
}

func (s *ServerTestSuite) TestGetPort() {
	assert.NotPanics(s.T(), func() {
		err := s.server.BootWithInterfaces(s.logger, s.router, s.tracer, &apiserver.Settings{})
		assert.NoError(s.T(), err)

		port, err := s.server.GetPort()
		assert.NoError(s.T(), err)
		assert.NotNil(s.T(), port)

		_, err = net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
		assert.NoError(s.T(), err, "could not establish a connection with server")
	})
}

func (s *ServerTestSuite) TestGetPort_Error() {
	assert.NotPanics(s.T(), func() {
		port, err := s.server.GetPort()
		assert.EqualError(s.T(), err, "could not get port. module is not yet booted")
		assert.Nil(s.T(), port)
	})
}

func (s *ServerTestSuite) TestBaseProfilingEndpoint() {
	s.T().Skip("profiling routes are added in Boot, cannot be tested here")

	assert.NotPanics(s.T(), func() {
		err := s.server.BootWithInterfaces(s.logger, s.router, s.tracer, &apiserver.Settings{})
		assert.NoError(s.T(), err)
	})

	httpRecorder := httptest.NewRecorder()

	assertRouteReturnsResponse(s.T(), s.router, httpRecorder, apiserver.BaseProfiling+"/", http.StatusOK)
}

func assertRouteReturnsResponse(t *testing.T, router *gin.Engine, httpRecorder *httptest.ResponseRecorder, route string, responseCode int) {
	var req *http.Request

	req, _ = http.NewRequest(http.MethodGet, route, nil)
	router.ServeHTTP(httpRecorder, req)

	assert.Equal(t, responseCode, httpRecorder.Code)
}
