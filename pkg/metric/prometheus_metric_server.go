package metric

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	prometheusDefaultRegistry = "default"
)

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

type metricsServer struct {
	kernel.EssentialBackgroundModule
	kernel.ServiceStage

	logger   log.Logger
	server   *http.Server
	listener net.Listener
}

func NewPrometheusMetricsServerModule(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	settings := getMetricSettings(config)
	if settings.WriterSettings.Prometheus.Api.Enabled {
		return NewPrometheusMetricServer(ctx, logger, settings.WriterSettings.Prometheus)
	}

	return nil, nil
}

func NewPrometheusMetricServer(ctx context.Context, logger log.Logger, settings PrometheusSettings) (kernel.Module, error) {
	registry, err := ProvideRegistry(ctx, prometheusDefaultRegistry)
	if err != nil {
		return nil, err
	}

	return NewMetricServerWithInterfaces(logger, registry, &settings)
}

func NewMetricServerWithInterfaces(logger log.Logger, registry *prometheus.Registry, settings *PrometheusSettings) (kernel.Module, error) {
	handler := http.NewServeMux()
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", settings.Api.Port),
		ReadTimeout:  settings.Api.Timeout.Read,
		WriteTimeout: settings.Api.Timeout.Write,
		IdleTimeout:  settings.Api.Timeout.Idle,
		Handler:      handler,
	}

	handler.Handle(settings.Api.Path, promhttp.InstrumentMetricHandler(
		registry, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}),
	))

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

	logger.Info("serving metrics on address %s", listener.Addr().String())

	return &metricsServer{
		logger:   logger,
		server:   server,
		listener: listener,
	}, nil
}

func (s *metricsServer) Run(ctx context.Context) error {
	var err error
	go func() {
		err = s.server.Serve(s.listener)
		if !errors.Is(http.ErrServerClosed, err) {
			s.logger.Error("Server closed unexpected: %w", err)

			return
		}
	}()

	for {
		select {
		case <-ctx.Done():
			err = s.server.Close()
			if err != nil {
				s.logger.Error("Server Close: %w", err)
			}

			s.logger.Info("leaving metrics server")
			return err
		}
	}
}

func (s *metricsServer) GetPort() (*int, error) {
	if s == nil {
		return nil, errors.New("metricsServer is nil, module is not yet booted")
	}

	if s.listener == nil {
		return nil, errors.New("could not get port. module is not yet booted")
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
