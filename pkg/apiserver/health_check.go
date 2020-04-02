package apiserver

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
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

	logger mon.Logger
	server *http.Server
}

func NewApiHealthCheck() *ApiHealthCheck {
	return &ApiHealthCheck{}
}

func (a *ApiHealthCheck) Boot(config cfg.Config, logger mon.Logger) error {
	settings := &ApiHealthCheckSettings{}
	config.UnmarshalKey("api.health", settings)

	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	return a.BootWithInterfaces(logger, r, settings)
}

func (a *ApiHealthCheck) BootWithInterfaces(logger mon.Logger, router *gin.Engine, s *ApiHealthCheckSettings) error {
	a.logger = logger

	router.Use(LoggingMiddleware(logger))
	router.GET(s.Path, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	addr := fmt.Sprintf(":%d", s.Port)

	a.server = &http.Server{
		Addr:    addr,
		Handler: router,
	}

	return nil
}

func (a *ApiHealthCheck) Run(ctx context.Context) error {
	go a.waitForStop(ctx)
	err := a.server.ListenAndServe()

	if err != http.ErrServerClosed {
		a.logger.Error(err, "api health check closed unexpected")
		return err
	}

	return nil
}

func (a *ApiHealthCheck) waitForStop(ctx context.Context) {
	<-ctx.Done()
	err := a.server.Close()

	if err != nil {
		a.logger.Error(err, "api health check close")
	}
}
