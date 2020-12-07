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

	logger   mon.Logger
	server   *http.Server
	listener net.Listener
}

func New(definer Definer) kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger mon.Logger) (kernel.Module, error) {
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

		definitions, err := definer(ctx, config, logger)
		if err != nil {
			return nil, fmt.Errorf("could not define routes: %w", err)
		}

		router.Use(RecoveryWithSentry(logger))
		router.Use(LoggingMiddleware(logger))

		buildRouter(definitions, router)

		return NewWithInterfaces(logger, router, tracer, settings)
	}
}

func NewWithInterfaces(logger mon.Logger, router *gin.Engine, tracer tracing.Tracer, s *Settings) (*ApiServer, error) {
	server := &http.Server{
		Addr:         ":" + s.Port,
		Handler:      tracer.HttpHandler(router),
		ReadTimeout:  s.TimeoutRead * time.Second,
		WriteTimeout: s.TimeoutWrite * time.Second,
		IdleTimeout:  s.TimeoutIdle * time.Second,
	}

	var err error
	var address = server.Addr
	var listener net.Listener

	if address == "" {
		address = ":http"
	}

	// open a port for the server already in this step so we can already start accepting connections
	// when this module is later run (see also issue #201)
	if listener, err = net.Listen("tcp", address); err != nil {
		return nil, err
	}

	logger.Infof("serving api requests on address %s", listener.Addr().String())

	apiServer := &ApiServer{
		logger:   logger,
		server:   server,
		listener: listener,
	}

	return apiServer, nil
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
	_, portStr, err := net.SplitHostPort(address)

	if err != nil {
		return nil, fmt.Errorf("could not get port from address %s: %w", address, err)
	}

	port, err := strconv.Atoi(portStr)

	if err != nil {
		return nil, fmt.Errorf("can not convert port string to int: %w", err)
	}

	return &port, nil
}
