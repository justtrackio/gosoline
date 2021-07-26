package apiserver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/gin-gonic/gin"
)

type ProfilingSettings struct {
	Port int `cfg:"port" default:"8091"`
}

type Profiling struct {
	kernel.BackgroundModule
	kernel.ServiceStage

	logger mon.Logger
	server *http.Server
}

func NewProfiling() kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger mon.Logger) (kernel.Module, error) {
		settings := &ProfilingSettings{}
		config.UnmarshalKey("profiling.api", settings)

		gin.SetMode(gin.ReleaseMode)
		router := gin.New()

		profiling := NewProfilingWithInterfaces(logger, router, settings)

		return profiling, nil
	}
}

func NewProfilingWithInterfaces(logger mon.Logger, router *gin.Engine, settings *ProfilingSettings) *Profiling {
	AddProfilingEndpoints(router)

	addr := fmt.Sprintf(":%d", settings.Port)

	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	return &Profiling{
		logger: logger,
		server: server,
	}
}

func (p *Profiling) Run(ctx context.Context) error {
	go p.waitForStop(ctx)
	err := p.server.ListenAndServe()

	if err != http.ErrServerClosed {
		p.logger.Error("api health check closed unexpected", err)
		return err
	}

	return nil
}

func (p *Profiling) waitForStop(ctx context.Context) {
	<-ctx.Done()
	err := p.server.Close()
	if err != nil {
		p.logger.Error("api health check close", err)
	}
}
