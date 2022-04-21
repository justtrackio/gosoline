package apiserver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

type ProfilingSettings struct {
	Port int `cfg:"port" default:"8091"`
}

type Profiling struct {
	kernel.BackgroundModule
	kernel.ServiceStage

	logger log.Logger
	server *http.Server
}

func NewProfiling() kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		settings := &ProfilingSettings{}
		config.UnmarshalKey("profiling.api", settings)

		gin.SetMode(gin.ReleaseMode)
		router := gin.New()

		profiling := NewProfilingWithInterfaces(logger, router, settings)

		return profiling, nil
	}
}

func NewProfilingWithInterfaces(logger log.Logger, router *gin.Engine, settings *ProfilingSettings) *Profiling {
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
		p.logger.Error("profiling api server closed unexpected", err)
		return err
	}

	return nil
}

func (p *Profiling) waitForStop(ctx context.Context) {
	<-ctx.Done()
	err := p.server.Close()
	if err != nil {
		p.logger.Error("profiling api server close", err)
	}
}
