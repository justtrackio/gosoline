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
	Port        string              `cfg:"port" default:"8080"`
	Mode        string              `cfg:"mode" default:"release" validate:"oneof=release debug test"`
	Compression CompressionSettings `cfg:"compression"`
	Timeout     TimeoutSettings     `cfg:"timeout"`
}

type TimeoutSettings struct {
	// read, write and idle timeouts. You need to give at least 1s as timeout.
	Read  time.Duration `cfg:"read" default:"60s" validate:"min=1000000000"`
	Write time.Duration `cfg:"write" default:"60s" validate:"min=1000000000"`
	Idle  time.Duration `cfg:"idle" default:"60s" validate:"min=1000000000"`
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
		var settings *Settings
		if config.IsSet("api_port") {
			// old code path, build the settings from various keys without validation or defaults (and with
			// no chance to configure compression)
			settings = &Settings{
				Port: config.GetString("api_port"),
				Mode: config.GetString("api_mode"),
				Compression: CompressionSettings{
					Level:         "none",
					Decompression: false,
					Exclude:       CompressionExcludeSettings{},
				},
				Timeout: TimeoutSettings{
					Read:  config.GetDuration("api_timeout_read") * time.Second,
					Write: config.GetDuration("api_timeout_write") * time.Second,
					Idle:  config.GetDuration("api_timeout_idle") * time.Second,
				},
			}
		} else {
			settings = &Settings{}
			config.UnmarshalKey("api", settings)
		}

		gin.SetMode(settings.Mode)

		tracer, err := tracing.ProvideTracer(config, logger)
		if err != nil {
			return nil, fmt.Errorf("can not create tracer: %w", err)
		}

		router := gin.New()
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

		if err := configureCompression(router, settings.Compression); err != nil {
			return nil, fmt.Errorf("could not configure compression: %w", err)
		}

		buildRouter(definitions, router)

		return NewWithInterfaces(logger, router, tracer, settings)
	}
}

func NewWithInterfaces(logger mon.Logger, router *gin.Engine, tracer tracing.Tracer, s *Settings) (*ApiServer, error) {
	server := &http.Server{
		Addr:         ":" + s.Port,
		Handler:      tracer.HttpHandler(router),
		ReadTimeout:  s.Timeout.Read,
		WriteTimeout: s.Timeout.Write,
		IdleTimeout:  s.Timeout.Idle,
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
