package apiserver

import (
	"context"
	"errors"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/gin-gonic/gin"
	"net"
	"net/http"
	"regexp"
	"strconv"
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
	kernel.ServiceStage

	logger       mon.Logger
	server       *http.Server
	listener     net.Listener
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

	router := gin.New()
	tracer := tracing.ProviderTracer(config, logger)

	AddProfilingEndpoints(router)

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	definitions := &Definitions{}
	a.defineRouter(config, logger, definitions)

	router.Use(RecoveryWithSentry(logger))
	router.Use(LoggingMiddleware(logger))

	buildRouter(definitions, router)

	return a.BootWithInterfaces(logger, router, tracer, settings)
}

func (a *ApiServer) BootWithInterfaces(logger mon.Logger, router *gin.Engine, tracer tracing.Tracer, s *Settings) error {
	a.logger = logger

	a.server = &http.Server{
		Addr:         ":" + s.Port,
		Handler:      tracer.HttpHandler(router),
		ReadTimeout:  s.TimeoutRead * time.Second,
		WriteTimeout: s.TimeoutWrite * time.Second,
		IdleTimeout:  s.TimeoutIdle * time.Second,
	}

	addr := a.server.Addr
	if addr == "" {
		addr = ":http"
	}

	// open a port for the server already in this step so we can already start accepting connections
	// when this module is later run (see also issue #201)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	a.listener = ln

	return nil
}

func (a *ApiServer) Run(ctx context.Context) error {
	go a.waitForStop(ctx)
	err := a.server.Serve(a.listener)

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

func (a *ApiServer) GetPort() (*int, error) {
	if a.listener == nil {
		return nil, errors.New("could not get port. module is not yet booted")
	}

	address := a.listener.Addr().String()
	pattern := regexp.MustCompile(`.+:(\d+)$`)
	matches := pattern.FindStringSubmatch(address)

	if len(matches) != 2 {
		return nil, fmt.Errorf("could not get port from address %s", address)
	}

	port, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil, err
	}

	return &port, nil
}
