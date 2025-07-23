package httpserver_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/clock"
	clockMocks "github.com/justtrackio/gosoline/pkg/clock/mocks"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/testify/assert"
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
	s.logger.EXPECT().WithFields(mock.AnythingOfType("log.Fields")).Return(s.logger)

	s.handler = httpserver.NewLoggingMiddlewareWithInterfaces(s.logger, httpserver.LoggingSettings{}, clock.Provider)
}

func (s *loggingMiddlewareTestSuite) TestSuccess() {
	ginCtx := buildRequest()

	s.logger.EXPECT().Info(matcher.Context, "%s %s %s", "GET", "path", "HTTP/1.1")

	s.handler(ginCtx)
}

func (s *loggingMiddlewareTestSuite) TestRequestCanceledError() {
	ginCtx := buildRequest()

	err := ginCtx.Error(context.Canceled)

	s.Require().Error(err)

	s.logger.EXPECT().Info(matcher.Context, "%s %s %s - request canceled: %s", "GET", "path", "HTTP/1.1", context.Canceled.Error())

	s.handler(ginCtx)
}

func (s *loggingMiddlewareTestSuite) TestEOFError() {
	ginCtx := buildRequest()

	err := ginCtx.Error(io.EOF)

	s.Require().Error(err)

	s.logger.EXPECT().Info(matcher.Context, "%s %s %s - connection error: %s", "GET", "path", "HTTP/1.1", io.EOF.Error())

	s.handler(ginCtx)
}

func (s *loggingMiddlewareTestSuite) TestBindError() {
	ginCtx := buildRequest()

	err := ginCtx.Error(&gin.Error{
		Err:  fmt.Errorf("failed to read body"),
		Type: gin.ErrorTypeBind,
	})

	s.Require().Error(err)

	s.logger.EXPECT().Warn(matcher.Context, "%s %s %s - bind error: %s", "GET", "path", "HTTP/1.1", "failed to read body")

	s.handler(ginCtx)
}

func (s *loggingMiddlewareTestSuite) TestRenderError() {
	ginCtx := buildRequest()

	err := ginCtx.Error(&gin.Error{
		Err:  fmt.Errorf("failed to write body"),
		Type: gin.ErrorTypeRender,
	})

	s.Require().Error(err)

	s.logger.EXPECT().Warn(matcher.Context, "%s %s %s - render error: %s", "GET", "path", "HTTP/1.1", "failed to write body")

	s.handler(ginCtx)
}

func (s *loggingMiddlewareTestSuite) TestDefaultError() {
	ginCtx := buildRequest()

	err := ginCtx.Error(fmt.Errorf("error"))

	s.Require().Error(err)

	s.logger.EXPECT().Error(matcher.Context, "%s %s %s: %w", "GET", "path", "HTTP/1.1", mock.AnythingOfType("*errors.errorString"))

	s.handler(ginCtx)
}

func TestLogFields(t *testing.T) {
	ginCtx := buildRequest()
	ginCtx.Request.Host = "host.io"
	ginCtx.Request.Header.Set("User-Agent", "test/User-Agent")
	ginCtx.Request.Header.Set("Referer", "referer")
	ginCtx.Request.Header.Set("X-Request-Id", "request123")
	ginCtx.Request.Header.Set("X-Session-Id", "session123")
	ginCtx.Request.RemoteAddr = "127.0.0.1:80"
	ginCtx.Status(http.StatusOK)

	_, err := ginCtx.Writer.WriteString("{}")
	assert.NoError(t, err)

	now := time.Date(2024, 12, 20, 12, 9, 3, 0, time.UTC)

	clock := clockMocks.NewClock(t)
	clock.EXPECT().Now().Return(now).Once()
	clock.EXPECT().Since(now).Return(250 * time.Millisecond).Once()

	logger := logMocks.NewLogger(t)

	expected := log.Fields{
		"bytes":            2,
		"client_ip":        "127.0.0.1",
		"host":             "host.io",
		"protocol":         "HTTP/1.1",
		"request_method":   "GET",
		"request_path":     "path",
		"request_path_raw": "path",
		"request_referer":  "referer",
		"request_time":     0.25,
		"request_query":    "foo=bar",
		"request_query_parameters": map[string]string{
			"foo": "bar",
		},
		"request_user_agent": "test/User-Agent",
		"scheme":             "https",
		"status":             200,
	}

	logger.EXPECT().WithFields(mock.AnythingOfType("log.Fields")).Run(func(fields log.Fields) {
		assert.Equal(t, expected, fields)
	}).Return(logger)

	logger.EXPECT().Info(matcher.Context, "%s %s %s", "GET", "path", "HTTP/1.1")

	handler := httpserver.NewLoggingMiddlewareWithInterfaces(logger, httpserver.LoggingSettings{}, clock)

	handler(ginCtx)
}

func TestLogEncodedRequestBody(t *testing.T) {
	ginCtx := buildRequest()
	ginCtx.Request.Body = io.NopCloser(strings.NewReader("{}"))

	logger := logMocks.NewLogger(t)

	logger.EXPECT().WithFields(mock.AnythingOfType("log.Fields")).Run(func(fields log.Fields) {
		requestBody, ok := fields["request_body"]
		assert.True(t, ok)
		assert.Equal(t, "e30=", requestBody)
	}).Return(logger)

	logger.EXPECT().Info(matcher.Context, "%s %s %s", "GET", "path", "HTTP/1.1")

	handler := httpserver.NewLoggingMiddlewareWithInterfaces(logger, httpserver.LoggingSettings{
		RequestBody:       true,
		RequestBodyBase64: true,
	}, clock.Provider)

	handler(ginCtx)
}

func buildRequest() *gin.Context {
	w := httptest.NewRecorder()

	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = &http.Request{
		Header: make(http.Header),
		Method: http.MethodGet,
		Proto:  "HTTP/1.1",
		URL: &url.URL{
			Scheme:   "https",
			Host:     "host.io",
			Path:     "path",
			RawPath:  "/path",
			RawQuery: "foo=bar",
		},
	}

	return ginCtx
}
