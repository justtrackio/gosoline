package db_repo

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

type HttpServerSettings struct {
	Enabled bool                      `cfg:"enabled" default:"false"`
	Port    string                    `cfg:"port" default:"8050"`
	Timeout HttpServerTimeoutSettings `cfg:"timeout"`
}

type HttpServerTimeoutSettings struct {
	// You need to give at least 1s as timeout.
	// Read timeout is the maximum duration for reading the entire request, including the body.
	Read time.Duration `cfg:"read" default:"60s" validate:"min=1000000000"`
	// Write timeout is the maximum duration before timing out writes of the response.
	Write time.Duration `cfg:"write" default:"60s" validate:"min=1000000000"`
	// Idle timeout is the maximum amount of time to wait for the next request when keep-alives are enabled
	Idle time.Duration `cfg:"idle" default:"60s" validate:"min=1000000000"`
}

type HttpServer struct {
	kernel.EssentialModule
	kernel.ServiceStage

	logger   log.Logger
	server   *http.Server
	listener net.Listener
}

func NewHttpServer() kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		var err error
		var handler *HttpHandler

		if handler, err = NewHttpHandler(ctx, config, logger); err != nil {
			return nil, fmt.Errorf("can not create http handler: %w", err)
		}

		settings := &HttpServerSettings{}
		config.UnmarshalKey("db_repo.http.server", settings)

		return NewHttpServerWithInterfaces(logger, handler, settings)
	}
}

func NewHttpServerWithInterfaces(logger log.Logger, handler *HttpHandler, settings *HttpServerSettings) (*HttpServer, error) {
	logger = logger.WithChannel("db-repo-http-server")

	server := &http.Server{
		Addr:         ":" + settings.Port,
		Handler:      handler,
		ReadTimeout:  settings.Timeout.Read,
		WriteTimeout: settings.Timeout.Write,
		IdleTimeout:  settings.Timeout.Idle,
	}

	var err error
	var listener net.Listener
	address := server.Addr

	if address == "" {
		address = ":http"
	}

	// open a port for the server already in this step so we can already start accepting connections
	// when this module is later run (see also issue #201)
	if listener, err = net.Listen("tcp", address); err != nil {
		return nil, err
	}

	logger.Info("serving db-repo requests on address %s", listener.Addr().String())

	module := &HttpServer{
		logger:   logger,
		server:   server,
		listener: listener,
	}

	return module, nil
}

func (h *HttpServer) Run(ctx context.Context) error {
	go h.waitForStop(ctx)

	err := h.server.Serve(h.listener)

	if err != http.ErrServerClosed {
		h.logger.Error("Server closed unexpected: %w", err)

		return err
	}

	return nil
}

func (h *HttpServer) waitForStop(ctx context.Context) {
	<-ctx.Done()
	err := h.server.Close()
	if err != nil {
		h.logger.Error("Server Close: %w", err)
	}

	h.logger.Info("leaving server")
}
