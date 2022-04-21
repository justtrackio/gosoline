package grpcserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/justtrackio/gosoline/pkg/cfg"
	protobuf "github.com/justtrackio/gosoline/pkg/grpcserver/proto/health/v1"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip"
)

const (
	grpcServerConfigKey = "grpc_server"
	grpcServiceChannel  = "grpc_service"
)

// Server a basic grpc.Server wrapper that allows also can have a basic Health Check functionality.
type Server struct {
	kernel.EssentialModule
	kernel.ServiceStage

	logger       log.Logger
	listener     net.Listener
	serverCtx    context.Context
	cancelFunc   context.CancelFunc
	server       *grpc.Server
	healthServer *healthServer
}

type Middleware grpc.UnaryServerInterceptor
type MiddlewareFactory func(logger log.Logger) Middleware

// New returns a kernel.ModuleFactory for the Server kernel.Module.
func New(name string, definer ServiceDefiner, middlewares ...MiddlewareFactory) kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		var (
			err         error
			definitions *Definitions
		)
		settings := &Settings{}
		config.UnmarshalKey(fmt.Sprintf("%s.%s", grpcServerConfigKey, name), settings)

		logger = logger.WithFields(log.Fields{
			"server_name": name,
		}).WithChannel(grpcServiceChannel)

		if definitions, err = definer(ctx, config, logger); err != nil {
			return nil, fmt.Errorf("could not define routes: %w", err)
		}

		return NewWithInterfaces(ctx, logger, definitions, settings, middlewares...)
	}
}

// NewWithInterfaces receives the interfaces required to create a Server.
func NewWithInterfaces(ctx context.Context, logger log.Logger, definitions *Definitions, s *Settings, middlewares ...MiddlewareFactory) (*Server, error) {
	var (
		hs         *healthServer
		cancelFunc context.CancelFunc
		serverCtx  = ctx
	)

	interceptors := []grpc.UnaryServerInterceptor{}
	for _, m := range middlewares {
		interceptors = append(interceptors,
			grpc.UnaryServerInterceptor(m(logger)))
	}

	options := []grpc.ServerOption{
		grpc.UnaryInterceptor(
			grpc_middleware.ChainUnaryServer(interceptors...),
		),
	}
	if s.Stats.Enabled {
		options = append(options, grpc.StatsHandler(NewStatsHandler(logger, s)))
	}

	server := grpc.NewServer(options...)
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", s.Port))
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %v", err)
	}

	if s.Health.Enabled {
		serverCtx, cancelFunc = context.WithCancel(ctx)
		hs = NewHealthServer(logger, cancelFunc)
		protobuf.RegisterHealthServer(server, hs)
		logger.Info("grpc_server enabled health-checks")
	}

	for _, def := range *definitions {
		if s.Health.Enabled && hs != nil && def.HealthCheckCallback != nil {
			hs.AddCallback(def.ServiceName, def.HealthCheckCallback)
		}
		err = def.Registrant(server)
		if err != nil {
			return nil, err
		}
	}

	logger.Info("grpc_server listens on address %s", listener.Addr().String())

	return &Server{
		logger:       logger,
		server:       server,
		listener:     listener,
		healthServer: hs,
		serverCtx:    serverCtx,
		cancelFunc:   cancelFunc,
	}, nil
}

// Run starts the Server kernel.Module, listens to the port configured and gracefully shuts down when the context is closed
// or if the HealthChecks are enabled when a service becomes unhealthy.
func (g *Server) Run(ctx context.Context) error {
	go g.waitForStop(ctx)

	err := g.server.Serve(g.listener)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		g.logger.Error("grpc_server closed unexpected: %w", err.Error())

		return err
	}

	return nil
}

// Addr Returns the net.Addr of the Server.
func (g *Server) Addr() net.Addr {
	return g.listener.Addr()
}

func (g *Server) waitForStop(ctx context.Context) {
	if g.cancelFunc != nil {
		defer g.cancelFunc()
	}

	select {
	case <-ctx.Done():
		g.logger.Info("stopping grpc_server due to canceled context")
	case <-g.serverCtx.Done():
		g.logger.Info("stopping grpc_server due to unhealthy service")
	}

	g.server.GracefulStop()
	g.logger.Info("leaving grpc_server")
}
