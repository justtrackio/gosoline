package httpserver_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type loggingMiddlewareTestSuite struct {
	suite.Suite

	logger *logMocks.Logger

	handler gin.HandlerFunc
}

func TestLoggingMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(loggingMiddlewareTestSuite))
}

func (s *loggingMiddlewareTestSuite) SetupTest() {
	s.logger = logMocks.NewLogger(s.T())
	s.logger.EXPECT().WithChannel("http").Return(s.logger)

	s.handler = httpserver.LoggingMiddleware(s.logger, httpserver.LoggingSettings{})
}

func (s *loggingMiddlewareTestSuite) TearDownTest() {
	s.logger.AssertExpectations(s.T())
}

func (s *loggingMiddlewareTestSuite) TestRequestCanceledError() {
	w := httptest.NewRecorder()

	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = &http.Request{
		Header: make(http.Header),
		URL:    &url.URL{},
	}

	_ = ginCtx.Error(context.Canceled)

	s.logger.EXPECT().WithContext(mock.Anything).Return(s.logger)
	s.logger.EXPECT().WithFields(mock.Anything).Return(s.logger)
	s.logger.EXPECT().Info("%s %s %s - request canceled - %w", "", "", "", mock.AnythingOfType("*gin.Error"))

	s.handler(ginCtx)
}

func (s *loggingMiddlewareTestSuite) TestEOFError() {
	w := httptest.NewRecorder()

	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = &http.Request{
		Header: make(http.Header),
		URL:    &url.URL{},
	}

	_ = ginCtx.Error(io.EOF)

	s.logger.EXPECT().WithContext(mock.Anything).Return(s.logger)
	s.logger.EXPECT().WithFields(mock.Anything).Return(s.logger)
	s.logger.EXPECT().Info("%s %s %s - connection error - %w", "", "", "", mock.AnythingOfType("*gin.Error"))

	s.handler(ginCtx)
}

// TODO finish coverage of middleware
