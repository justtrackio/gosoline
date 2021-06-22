package apiserver_test

import (
	"github.com/applike/gosoline/pkg/apiserver"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewApiHealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ginEngine := gin.New()
	logger := logMocks.NewLoggerMockedAll()

	apiserver.NewApiHealthCheckWithInterfaces(logger, ginEngine, &apiserver.ApiHealthCheckSettings{
		Path: "/health",
	})

	httpRecorder := httptest.NewRecorder()
	assertRouteReturnsResponse(t, ginEngine, httpRecorder, "/health", http.StatusOK)
}
