package httpserver_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/kernel"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
)

func HealthCheckerMock() kernel.HealthCheckResult {
	return make(kernel.HealthCheckResult, 0)
}

func TestNewApiHealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ginEngine := gin.New()
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))

	httpserver.NewHealthCheckWithInterfaces(logger, ginEngine, HealthCheckerMock, &httpserver.HealthCheckSettings{
		Path: "/health",
	})

	httpRecorder := httptest.NewRecorder()
	assertRouteReturnsResponse(t, ginEngine, httpRecorder, "/health", http.StatusOK)
}
