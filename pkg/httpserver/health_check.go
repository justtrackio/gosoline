package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/dx"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

func init() {
	dx.RegisterRandomizablePortSetting("httpserver.health-check.port")
}

type HealthCheckSettings struct {
	Port int    `cfg:"port" default:"8090"`
	Path string `cfg:"path" default:"/health"`
}

type ApiHealthCheck struct {
	kernel.BackgroundModule
	kernel.ServiceStage

	logger log.Logger
	server *http.Server
}

func NewHealthCheck() kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		settings := &HealthCheckSettings{}
		config.UnmarshalKey("httpserver.health-check", settings)

		gin.SetMode(gin.ReleaseMode)
		router := gin.New()

		healthChecker, err := kernel.GetHealthChecker(ctx)
		if err != nil {
			return nil, fmt.Errorf("can not get health checker: %w", err)
		}

		return NewHealthCheckWithInterfaces(logger, router, healthChecker, settings), nil
	}
}

func NewHealthCheckWithInterfaces(logger log.Logger, router *gin.Engine, healthChecker kernel.HealthChecker, settings *HealthCheckSettings) *ApiHealthCheck {
	logger = logger.WithChannel("httpserver-health-check")

	router.Use(LoggingMiddleware(logger, LoggingSettings{}))
	router.GET(settings.Path, buildHealthCheckHandler(logger, healthChecker))

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", settings.Port),
		Handler: router,
	}

	return &ApiHealthCheck{
		logger: logger,
		server: server,
	}
}

func (a *ApiHealthCheck) Run(ctx context.Context) error {
	go a.waitForStop(ctx)

	err := a.server.ListenAndServe()

	if !errors.Is(err, http.ErrServerClosed) {
		a.logger.Error("api health check closed unexpected: %w", err)

		return err
	}

	return nil
}

func (a *ApiHealthCheck) waitForStop(ctx context.Context) {
	<-ctx.Done()

	if err := a.server.Close(); err != nil {
		a.logger.Error("server health check close: %w", err)
	}
}

func buildHealthCheckHandler(logger log.Logger, healthChecker kernel.HealthChecker) func(c *gin.Context) {
	return func(c *gin.Context) {
		result := healthChecker()

		if result.IsHealthy() {
			c.JSON(http.StatusOK, gin.H{})

			return
		}

		if result.Err() != nil {
			ctx := c.Request.Context()
			logger.WithContext(ctx).Error("encountered an error during the health check: %", result.Err())
		}

		resp := gin.H{}
		for _, module := range result.GetUnhealthy() {
			if module.Err != nil {
				resp[module.Name] = module.Err.Error()
			} else {
				resp[module.Name] = "unhealthy"
			}
		}

		c.JSON(http.StatusInternalServerError, resp)
	}
}
