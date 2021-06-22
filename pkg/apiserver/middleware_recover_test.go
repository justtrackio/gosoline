package apiserver_test

import (
	"errors"
	"github.com/applike/gosoline/pkg/apiserver"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/unix"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecoveryWithSentryCaseNil(t *testing.T) {
	gin.SetMode(gin.TestMode)

	loggerMock := logMocks.NewLoggerMockedAll()

	r := gin.New()
	r.Use(apiserver.RecoveryWithSentry(loggerMock))

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

	loggerMock := logMocks.NewLoggerMockedAll()

	r := gin.New()
	r.Use(apiserver.RecoveryWithSentry(loggerMock))
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

	loggerMock := logMocks.NewLoggerMockedAll()

	r := gin.New()
	r.Use(apiserver.RecoveryWithSentry(loggerMock))
	r.Use(func(c *gin.Context) {
		err := apiserver.ResponseBodyWriterError{Err: unix.EPIPE}
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

	loggerMock := logMocks.NewLoggerMockedAll()

	r := gin.New()
	r.Use(apiserver.RecoveryWithSentry(loggerMock))
	r.Use(func(c *gin.Context) {
		err := apiserver.ResponseBodyWriterError{Err: errors.New("an error")}
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

	loggerMock := logMocks.NewLoggerMockedAll()

	r := gin.New()
	r.Use(apiserver.RecoveryWithSentry(loggerMock))
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
