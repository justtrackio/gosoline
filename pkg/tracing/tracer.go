package tracing

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"net/http"
	"sync"
)

//go:generate mockery -name=Tracer
type Tracer interface {
	HttpHandler(h http.Handler) http.Handler
	StartSpan(name string) (context.Context, Span)
	StartSpanFromContext(ctx context.Context, name string) (context.Context, Span)
	StartSubSpan(ctx context.Context, name string) (context.Context, Span)
}

type TracerSettings struct {
	Provider     string                `cfg:"provider" default:"xray" validate:"required"`
	Enabled      bool                  `cfg:"enabled" default:"false"`
	AddressType  string                `cfg:"addr_type" default:"local" validate:"required"`
	AddressValue string                `cfg:"add_value" default:""`
	Sampling     SamplingConfiguration `cfg:"sampling"`
}

var tracerContainer = struct {
	sync.Mutex
	instance Tracer
}{}

func ProviderTracer(config cfg.Config, logger mon.Logger) Tracer {
	tracerContainer.Lock()
	defer tracerContainer.Unlock()

	if tracerContainer.instance != nil {
		return tracerContainer.instance
	}

	tracerContainer.instance = NewTracer(config, logger)

	return tracerContainer.instance
}

func NewTracer(config cfg.Config, logger mon.Logger) Tracer {
	settings := &TracerSettings{}
	config.UnmarshalKey("tracing", settings)

	if !settings.Enabled {
		return NewNoopTracer()
	}

	if _, ok := providers[settings.Provider]; !ok {
		err := fmt.Errorf("no tracing provider found for name %s", settings.Provider)
		logger.Fatalf(err, err.Error())
	}

	provider := providers[settings.Provider]

	return provider(config, logger)
}
