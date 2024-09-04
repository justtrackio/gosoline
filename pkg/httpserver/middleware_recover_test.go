package httpserver_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/unix"
)

func TestRecoveryWithSentryCaseNil(t *testing.T) {
	gin.SetMode(gin.TestMode)

	loggerMock := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))

	r := gin.New()
	r.Use(httpserver.RecoveryWithSentry(loggerMock))

	var req *http.Request
	httpRecorder := httptest.NewRecorder()

	req, _ = http.NewRequest(http.MethodGet, "/some/route", nil)

	assert.NotPanics(t, func() {
		r.ServeHTTP(httpRecorder, req)
	})
	loggerMock.AssertNumberOfCalls(t, "Warn", 0)
	loggerMock.AssertNumberOfCalls(t, "Error", 0)
}

func TestRecoveryWithSentryCaseError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	loggerMock := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))

	r := gin.New()
	r.Use(httpserver.RecoveryWithSentry(loggerMock))
	r.Use(func(c *gin.Context) {
		err := http.ErrServerClosed
		panic(err)
	})

	var req *http.Request
	httpRecorder := httptest.NewRecorder()

	req, _ = http.NewRequest(http.MethodGet, "/some/route", nil)

	assert.NotPanics(t, func() {
		r.ServeHTTP(httpRecorder, req)
	})
	loggerMock.AssertNumberOfCalls(t, "Warn", 0)
	loggerMock.AssertNumberOfCalls(t, "Error", 1)
}

func TestRecoveryWithSentryCaseResponseBodyWriterAndConnectionErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	loggerMock := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))

	r := gin.New()
	r.Use(httpserver.RecoveryWithSentry(loggerMock))
	r.Use(func(c *gin.Context) {
		err := httpserver.ResponseBodyWriterError{Err: unix.EPIPE}
		panic(err)
	})

	var req *http.Request
	httpRecorder := httptest.NewRecorder()

	req, _ = http.NewRequest(http.MethodGet, "/some/route", nil)

	assert.NotPanics(t, func() {
		r.ServeHTTP(httpRecorder, req)
	})
	loggerMock.AssertNumberOfCalls(t, "Warn", 1)
	loggerMock.AssertNumberOfCalls(t, "Error", 0)
}

func TestRecoveryWithSentryCaseResponseBodyWriterErrorButNotConnectionError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	loggerMock := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))

	r := gin.New()
	r.Use(httpserver.RecoveryWithSentry(loggerMock))
	r.Use(func(c *gin.Context) {
		err := httpserver.ResponseBodyWriterError{Err: errors.New("an error")}
		panic(err)
	})

	var req *http.Request
	httpRecorder := httptest.NewRecorder()

	req, _ = http.NewRequest(http.MethodGet, "/some/route", nil)

	assert.NotPanics(t, func() {
		r.ServeHTTP(httpRecorder, req)
	})
	loggerMock.AssertNumberOfCalls(t, "Warn", 0)
	loggerMock.AssertNumberOfCalls(t, "Error", 1)
}

func TestRecoveryWithSentryCaseString(t *testing.T) {
	gin.SetMode(gin.TestMode)

	loggerMock := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))

	r := gin.New()
	r.Use(httpserver.RecoveryWithSentry(loggerMock))
	r.Use(func(c *gin.Context) {
		panic("Panic to test recovery")
	})

	var req *http.Request
	httpRecorder := httptest.NewRecorder()

	req, _ = http.NewRequest(http.MethodGet, "/some/route", nil)

	assert.NotPanics(t, func() {
		r.ServeHTTP(httpRecorder, req)
	})
	loggerMock.AssertNumberOfCalls(t, "Warn", 0)
	loggerMock.AssertNumberOfCalls(t, "Error", 1)
}
