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
	"time"

	"github.com/gin-contrib/location"
	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
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

func NewWithInterfaces(logger log.Logger, router *gin.Engine, tracer tracing.Instrumentor, s *Settings) (*HttpServer, error) {
	server := &http.Server{
		Addr:         ":" + s.Port,
		Handler:      tracer.HttpHandler(router),
		ReadTimeout:  s.Timeout.Read,
		WriteTimeout: s.Timeout.Write,
		IdleTimeout:  s.Timeout.Idle,
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

	logger.Info("serving api requests on address %s", listener.Addr().String())

	apiServer := &HttpServer{
		logger:   logger,
		server:   server,
		listener: listener,
	}

	return apiServer, nil
}

func (a *HttpServer) Run(ctx context.Context) error {
	go a.waitForStop(ctx)

	err := a.server.Serve(a.listener)

	if !errors.Is(err, http.ErrServerClosed) {
		a.logger.Error("Server closed unexpected: %w", err)

		return err
	}

	return nil
}

func (a *HttpServer) waitForStop(ctx context.Context) {
	<-ctx.Done()
	err := a.server.Close()
	if err != nil {
		a.logger.Error("Server Close: %w", err)
	}

	a.logger.Info("leaving api")
}

func (a *HttpServer) GetPort() (*int, error) {
	if a == nil {
		return nil, errors.New("apiServer is nil, module is not yet booted")
	}

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
