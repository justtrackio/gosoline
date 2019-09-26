package apiserver_test

import (
	"github.com/applike/gosoline/pkg/apiserver"
	"github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewApiHealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ginEngine := gin.New()
	logger := mocks.NewLoggerMockedAll()

	apiHealthCheck := apiserver.NewApiHealthCheck()
	err := apiHealthCheck.BootWithInterfaces(logger, ginEngine, &apiserver.ApiHealthCheckSettings{
		Path: "/health",
	})

	assert.NoError(t, err)

	httpRecorder := httptest.NewRecorder()

	assertRouteReturnsResponse(t, ginEngine, httpRecorder, "/health", http.StatusOK)
}
