package apiserver_test

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/apiserver"
	"github.com/applike/gosoline/pkg/log"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
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
	logger log.Logger
	router *gin.Engine
	tracer tracing.Tracer
	server *apiserver.ApiServer
}

func TestServerTestSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}

func (s *ServerTestSuite) SetupTest() {
	s.logger = logMocks.NewLoggerMockedAll()

	gin.SetMode(gin.TestMode)
	s.router = gin.New()

	tracer := new(tracingMocks.Tracer)
	tracer.On("HttpHandler", s.router).Return(s.router)
	s.tracer = tracer

	server, err := apiserver.NewWithInterfaces(s.logger, s.router, s.tracer, &apiserver.Settings{})
	s.NoError(err)

	s.server = server
}

func (s *ServerTestSuite) TestLifecycle_Cancel() {
	ctx, cancel := context.WithCancel(context.Background())
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

		_, err = net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
		s.NoError(err, "could not establish a connection with server")
	})
}

func (s *ServerTestSuite) TestBaseProfilingEndpoint() {
	s.NotPanics(func() {
		apiserver.AddProfilingEndpoints(s.router)
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
