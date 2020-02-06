package apiserver

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

type Settings struct {
	Port         string
	Mode         string
	TimeoutRead  time.Duration
	TimeoutWrite time.Duration
	TimeoutIdle  time.Duration
}

type ApiServer struct {
	kernel.EssentialModule

	logger       mon.Logger
	server       *http.Server
	defineRouter Define
}

func New(definer Define) *ApiServer {
	return &ApiServer{
		defineRouter: definer,
	}
}

func NewOnlyHealthRoute() *ApiServer {
	return &ApiServer{
		defineRouter: func(_ cfg.Config, _ mon.Logger, _ *Definitions) {},
	}
}

func (a *ApiServer) Boot(config cfg.Config, logger mon.Logger) error {
	settings := &Settings{
		Port:         config.GetString("api_port"),
		Mode:         config.GetString("api_mode"),
		TimeoutRead:  config.GetDuration("api_timeout_read"),
		TimeoutWrite: config.GetDuration("api_timeout_write"),
		TimeoutIdle:  config.GetDuration("api_timeout_idle"),
	}

	gin.SetMode(settings.Mode)

	r := gin.New()
	tracer := tracing.ProviderTracer(config, logger)

	return a.BootWithInterfaces(config, logger, r, tracer, settings)
}

func (a *ApiServer) BootWithInterfaces(config cfg.Config, logger mon.Logger, router *gin.Engine, tracer tracing.Tracer, s *Settings) error {
	addProfilingEndpoints(router)

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	a.logger = logger

	definitions := &Definitions{}
	a.defineRouter(config, logger, definitions)

	router.Use(RecoveryWithSentry(logger))
	router.Use(LoggingMiddleware(logger))

	buildRouter(definitions, router)

	a.server = &http.Server{
		Addr:         ":" + s.Port,
		Handler:      tracer.HttpHandler(router),
		ReadTimeout:  s.TimeoutRead * time.Second,
		WriteTimeout: s.TimeoutWrite * time.Second,
		IdleTimeout:  s.TimeoutIdle * time.Second,
	}

	return nil
}

func (a *ApiServer) Run(ctx context.Context) error {
	go a.waitForStop(ctx)
	err := a.server.ListenAndServe()

	if err != http.ErrServerClosed {
		a.logger.Error(err, "Server closed unexpected")
		return err
	}

	return nil
}

func (a *ApiServer) waitForStop(ctx context.Context) {
	<-ctx.Done()
	err := a.server.Close()

	if err != nil {
		a.logger.Error(err, "Server Close")
	}

	a.logger.Info("leaving api")
}
