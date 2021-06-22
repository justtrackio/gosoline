package apiserver

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/log"
	"github.com/gin-gonic/gin"
	"net/http"
)

type ApiHealthCheckSettings struct {
	Port int    `cfg:"port" default:"8090"`
	Path string `cfg:"path" default:"/health"`
}

type ApiHealthCheck struct {
	kernel.BackgroundModule
	kernel.ServiceStage

	logger log.Logger
	server *http.Server
}

func NewApiHealthCheck() kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		settings := &ApiHealthCheckSettings{}
		config.UnmarshalKey("api.health", settings)

		gin.SetMode(gin.ReleaseMode)
		router := gin.New()

		healthCheck := NewApiHealthCheckWithInterfaces(logger, router, settings)

		return healthCheck, nil
	}
}

func NewApiHealthCheckWithInterfaces(logger log.Logger, router *gin.Engine, settings *ApiHealthCheckSettings) *ApiHealthCheck {
	router.Use(LoggingMiddleware(logger))
	router.GET(settings.Path, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	addr := fmt.Sprintf(":%d", settings.Port)

	server := &http.Server{
		Addr:    addr,
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

	if err != http.ErrServerClosed {
		a.logger.Error("api health check closed unexpected: %w", err)
		return err
	}

	return nil
}

func (a *ApiHealthCheck) waitForStop(ctx context.Context) {
	<-ctx.Done()
	err := a.server.Close()

	if err != nil {
		a.logger.Error("api health check close: %w", err)
	}
}
