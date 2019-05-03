package apiserver_test

import (
	"github.com/applike/gosoline/pkg/apiserver"
	monMocks "github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecoveryWithSentryCaseNil(t *testing.T) {
	gin.SetMode(gin.TestMode)

	loggerMock := monMocks.NewLoggerMockedAll()

	r := gin.New()
	r.Use(apiserver.RecoveryWithSentry(loggerMock))

	var req *http.Request
	httpRecorder := httptest.NewRecorder()

	req, _ = http.NewRequest(http.MethodGet, "/some/route", nil)

	assert.NotPanics(t, func() {
		r.ServeHTTP(httpRecorder, req)
	})
}

func TestRecoveryWithSentryCaseError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	loggerMock := monMocks.NewLoggerMockedAll()

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
}

func TestRecoveryWithSentryCaseDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)

	loggerMock := monMocks.NewLoggerMockedAll()

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
}
