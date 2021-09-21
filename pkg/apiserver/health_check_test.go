package apiserver_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/apiserver"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
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
