package httpserver_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/tracing"
	tracingMocks "github.com/justtrackio/gosoline/pkg/tracing/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ServerTestSuite struct {
	suite.Suite
	logger              log.Logger
	router              *gin.Engine
	tracingInstrumentor tracing.Instrumentor
	server              *httpserver.HttpServer
}

func TestServerTestSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}

func (s *ServerTestSuite) SetupTest() {
	s.logger = logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))

	gin.SetMode(gin.TestMode)
	s.router = gin.New()

	tracingInstrumentor := tracingMocks.NewInstrumentor(s.T())
	tracingInstrumentor.EXPECT().HttpHandler(s.router).Return(s.router)
	s.tracingInstrumentor = tracingInstrumentor

	server, err := httpserver.NewWithInterfaces(s.T().Context(), s.logger, s.router, s.tracingInstrumentor, &httpserver.Settings{})
	s.NoError(err)

	s.server = server
}

func (s *ServerTestSuite) TestLifecycle_Cancel() {
	ctx, cancel := context.WithCancel(s.T().Context())
	cancel()

	s.NotPanics(func() {
		err := s.server.Run(ctx)
		s.NoError(err)
	})
}

func (s *ServerTestSuite) TestGetPort() {
	s.NotPanics(func() {
		port, err := s.server.GetPort()
		s.NoError(err)
		s.NotNil(port)

		address := net.JoinHostPort("127.0.0.1", fmt.Sprint(*port))
		_, err = net.Dial("tcp", address)
		s.NoError(err, "could not establish a connection with server")
	})
}

func (s *ServerTestSuite) TestBaseProfilingEndpoint() {
	s.NotPanics(func() {
		httpserver.AddProfilingEndpoints(s.router)
	})

	httpRecorder := httptest.NewRecorder()
	assertRouteReturnsResponse(s.T(), s.router, httpRecorder, httpserver.BaseProfiling+"/", http.StatusOK)
}

func assertRouteReturnsResponse(t *testing.T, router *gin.Engine, httpRecorder *httptest.ResponseRecorder, route string, responseCode int) {
	var req *http.Request
	var err error

	req, err = http.NewRequest(http.MethodGet, route, http.NoBody)
	assert.NoError(t, err, "could not create request for route %s", route)
	router.ServeHTTP(httpRecorder, req)

	assert.Equal(t, responseCode, httpRecorder.Code)
}
