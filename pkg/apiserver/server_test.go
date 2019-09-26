package apiserver_test

import (
	"context"
	"github.com/applike/gosoline/pkg/apiserver"
	"github.com/applike/gosoline/pkg/cfg"
	configMocks "github.com/applike/gosoline/pkg/cfg/mocks"
	"github.com/applike/gosoline/pkg/mon"
	monMocks "github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/applike/gosoline/pkg/tracing"
	tracingMocks "github.com/applike/gosoline/pkg/tracing/mocks"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func getMocks() (*monMocks.Logger, *configMocks.Config, *gin.Engine, tracing.Tracer) {
	loggingMock := monMocks.NewLoggerMockedAll()
	configMock := new(configMocks.Config)
	router := gin.New()

	tracer := new(tracingMocks.Tracer)
	tracer.On("HttpHandler", router).Return(router)

	return loggingMock, configMock, router, tracer
}

func TestBaseProfilingEndpoint(t *testing.T) {
	ginEngine := setup(t)
	httpRecorder := httptest.NewRecorder()

	assertRouteReturnsResponse(t, ginEngine, httpRecorder, apiserver.BaseProfiling+"/", http.StatusOK)
}

func TestApiServer_Lifecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	loggingMock, configMock, router, tracer := getMocks()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	assert.NotPanics(t, func() {
		a := apiserver.New(func(config cfg.Config, logger mon.Logger, definitions *apiserver.Definitions) {})

		a.BootWithInterfaces(configMock, loggingMock, router, tracer, &apiserver.Settings{})
		a.Run(ctx)
	})
}

func setup(t *testing.T) (ginEngine *gin.Engine) {
	gin.SetMode(gin.TestMode)
	loggingMock, configMock, router, tracer := getMocks()

	definer := func(configMock cfg.Config, logger mon.Logger, definitions *apiserver.Definitions) {
		ginEngine = router
	}

	a := apiserver.New(definer)

	assert.NotPanics(t, func() {
		a.BootWithInterfaces(configMock, loggingMock, router, tracer, &apiserver.Settings{})
	})

	return ginEngine
}

func assertRouteReturnsResponse(t *testing.T, router *gin.Engine, httpRecorder *httptest.ResponseRecorder, route string, responseCode int) {
	var req *http.Request

	req, _ = http.NewRequest(http.MethodGet, route, nil)
	router.ServeHTTP(httpRecorder, req)

	assert.Equal(t, responseCode, httpRecorder.Code)
}
