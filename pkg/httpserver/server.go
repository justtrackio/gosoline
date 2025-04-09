package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-contrib/location"
	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/tracing"
)

type ServerMetadata struct {
	Name     string            `json:"name"`
	Handlers []HandlerMetadata `json:"handlers"`
}

// HandlerMetadata stores the Path and Method of this Handler.
type HandlerMetadata struct {
	// Method is the route method of this Handler.
	Method string `json:"method"`
	// Path is the route path ot this handler.
	Path string `json:"path"`
}

// Settings structure for an API server.
type Settings struct {
	// Port the API listens to.
	Port string `cfg:"port" default:"8080"`
	// Mode is either debug, release, test.
	Mode string `cfg:"mode" default:"release" validate:"oneof=release debug test"`
	// Compression settings.
	Compression CompressionSettings `cfg:"compression"`
	// Timeout settings.
	Timeout TimeoutSettings `cfg:"timeout"`
	// Logging settings
	Logging LoggingSettings `cfg:"logging"`
}

// TimeoutSettings configures IO timeouts.
type TimeoutSettings struct {
	// You need to give at least 1s as timeout.
	// Read timeout is the maximum duration for reading the entire request, including the body.
	Read time.Duration `cfg:"read" default:"60s" validate:"min=1000000000"`
	// Write timeout is the maximum duration before timing out writes of the response.
	Write time.Duration `cfg:"write" default:"60s" validate:"min=1000000000"`
	// Idle timeout is the maximum amount of time to wait for the next request when keep-alives are enabled
	Idle time.Duration `cfg:"idle" default:"60s" validate:"min=1000000000"`
	// Drain timeout is the maximum amount of time to wait after receiving the kernel stop signal and actually shutting down the server
	Drain time.Duration `cfg:"drain" default:"0" validate:"min=0"`
	// Shutdown timeout is the maximum amount of time to wait for serving active requests before stopping the server
	Shutdown time.Duration `cfg:"shutdown" default:"60s" validate:"min=1000000000"`
}

type LoggingSettings struct {
	RequestBody       bool `cfg:"request_body"`
	RequestBodyBase64 bool `cfg:"request_body_base64"`
}

type HttpServer struct {
	kernel.EssentialModule
	kernel.ServiceStage

	logger   log.Logger
	server   *http.Server
	listener net.Listener
	settings *Settings
	healthy  atomic.Bool
}

func New(name string, definer Definer) kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		key := fmt.Sprintf("httpserver.%s", name)
		settings := &Settings{}
		config.UnmarshalKey(key, settings)

		return NewWithSettings(name, definer, settings)(ctx, config, logger)
	}
}

func NewWithSettings(name string, definer Definer, settings *Settings) kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		channel := fmt.Sprintf("httpserver-%s", name)
		logger = logger.WithChannel(channel)

		gin.SetMode(settings.Mode)

		var err error
		var tracingInstrumentor tracing.Instrumentor
		var definitions *Definitions
		var compressionMiddlewares []gin.HandlerFunc
		var healthChecker kernel.HealthChecker

		if tracingInstrumentor, err = tracing.ProvideInstrumentor(ctx, config, logger); err != nil {
			return nil, fmt.Errorf("can not create tracingInstrumentor: %w", err)
		}

		metricMiddleware, setupMetricMiddleware := NewMetricMiddleware(name)

		if compressionMiddlewares, err = configureCompression(settings.Compression); err != nil {
			return nil, fmt.Errorf("could not configure compression: %w", err)
		}

		router := gin.New()
		router.Use(metricMiddleware)
		router.Use(LoggingMiddleware(logger, settings.Logging))
		router.Use(compressionMiddlewares...)
		router.Use(RecoveryWithSentry(logger))
		router.Use(location.Default())

		if healthChecker, err = kernel.GetHealthChecker(ctx); err != nil {
			return nil, fmt.Errorf("can not get health checker: %w", err)
		}
		router.GET("/health", buildHealthCheckHandler(logger, healthChecker))

		if definitions, err = definer(ctx, config, logger.WithChannel("handler")); err != nil {
			return nil, fmt.Errorf("could not define routes: %w", err)
		}

		definitionList := buildRouter(definitions, router)
		setupMetricMiddleware(definitionList)

		if err = appendMetadata(ctx, name, router); err != nil {
			return nil, fmt.Errorf("can not append metadata: %w", err)
		}

		return NewWithInterfaces(logger, router, tracingInstrumentor, settings)
	}
}

func NewWithInterfaces(logger log.Logger, router *gin.Engine, tracer tracing.Instrumentor, settings *Settings) (*HttpServer, error) {
	server := &http.Server{
		Addr:         ":" + settings.Port,
		Handler:      tracer.HttpHandler(router),
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

	logger.Info("serving httpserver requests on address %s", listener.Addr().String())

	apiServer := &HttpServer{
		logger:   logger,
		server:   server,
		listener: listener,
		settings: settings,
	}

	return apiServer, nil
}

func (s *HttpServer) IsHealthy(ctx context.Context) (bool, error) {
	return s.healthy.Load(), nil
}

func (s *HttpServer) Run(ctx context.Context) error {
	go s.waitForStop(ctx)

	err := s.server.Serve(s.listener)

	if !errors.Is(err, http.ErrServerClosed) {
		s.logger.Error("server closed unexpected: %w", err)

		return err
	}

	s.logger.Info("leaving httpserver")

	return nil
}

func (s *HttpServer) waitForStop(ctx context.Context) {
	s.healthy.Store(true)
	<-ctx.Done()
	s.healthy.Store(false)

	s.logger.Info("waiting %s until shutting down the server", s.settings.Timeout.Drain)

	t := clock.NewRealTimer(s.settings.Timeout.Drain)
	defer t.Stop()
	<-t.Chan()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.settings.Timeout.Shutdown)
	defer cancel()

	s.logger.Info("trying to gracefully shutdown httpserver")

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		s.logger.Error("server shutdown: %w", err)
	}
}

func (s *HttpServer) GetPort() (*int, error) {
	if s == nil {
		return nil, errors.New("httpserver is nil, module is not yet running")
	}

	if s.listener == nil {
		return nil, errors.New("could not get port. module is not yet running")
	}

	address := s.listener.Addr().String()
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

func appendMetadata(ctx context.Context, name string, router *gin.Engine) error {
	var err error
	var metadata *appctx.Metadata

	serverMetadata := ServerMetadata{
		Name: name,
	}

	routes := router.Routes()
	slices.SortFunc(routes, func(a, b gin.RouteInfo) int {
		return strings.Compare(a.Path+a.Method, b.Path+a.Method)
	})

	for _, route := range routes {
		serverMetadata.Handlers = append(serverMetadata.Handlers, HandlerMetadata{
			Method: route.Method,
			Path:   route.Path,
		})
	}

	if metadata, err = appctx.ProvideMetadata(ctx); err != nil {
		return fmt.Errorf("can not access appctx metadata: %w", err)
	}

	if err = metadata.Append("httpservers", serverMetadata); err != nil {
		return fmt.Errorf("can not append httpserver routes to appctx metadata: %w", err)
	}

	return nil
}
