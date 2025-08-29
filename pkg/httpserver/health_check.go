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

type HttpServerHealthCheck struct {
	kernel.BackgroundModule
	kernel.EssentialStage

	logger   log.Logger
	server   *http.Server
	settings *HealthCheckSettings
}

func NewHealthCheck() kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		settings := &HealthCheckSettings{}
		if err := config.UnmarshalKey("httpserver.health-check", settings); err != nil {
			return nil, fmt.Errorf("failed to unmarshal health check settings: %w", err)
		}

		gin.SetMode(gin.ReleaseMode)
		router := gin.New()

		healthChecker, err := kernel.GetHealthChecker(ctx)
		if err != nil {
			return nil, fmt.Errorf("can not get health checker: %w", err)
		}

		return NewHealthCheckWithInterfaces(logger, router, healthChecker, settings), nil
	}
}

func NewHealthCheckWithInterfaces(logger log.Logger, router *gin.Engine, healthChecker kernel.HealthChecker, settings *HealthCheckSettings) *HttpServerHealthCheck {
	logger = logger.WithChannel("httpserver-health-check")

	router.Use(LoggingMiddleware(logger, LoggingSettings{}))
	router.GET(settings.Path, buildHealthCheckHandler(logger, healthChecker))

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", settings.Port),
		Handler: router,
	}

	return &HttpServerHealthCheck{
		logger:   logger,
		server:   server,
		settings: settings,
	}
}

func (a *HttpServerHealthCheck) Run(ctx context.Context) error {
	go a.waitForStop(ctx)

	err := a.server.ListenAndServe()

	if !errors.Is(err, http.ErrServerClosed) {
		a.logger.Error(ctx, "server check closed unexpected: %w", err)

		return err
	}

	a.logger.Info(ctx, "leaving httpserver health check")

	return nil
}

func (s *HttpServerHealthCheck) waitForStop(ctx context.Context) {
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.settings.Timeout.Shutdown)
	defer cancel()

	s.logger.Info(shutdownCtx, "trying to gracefully shutdown httpserver health check")

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		s.logger.Error(shutdownCtx, "server shutdown: %w", err)
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
			logger.Error(ctx, "encountered an error during the health check: %", result.Err())
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
