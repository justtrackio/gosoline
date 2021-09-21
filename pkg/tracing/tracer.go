package tracing

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

//go:generate mockery --name Tracer
type Tracer interface {
	HttpHandler(h http.Handler) http.Handler
	StartSpan(name string) (context.Context, Span)
	StartSpanFromContext(ctx context.Context, name string) (context.Context, Span)
	StartSubSpan(ctx context.Context, name string) (context.Context, Span)
}

type TracerSettings struct {
	Provider                    string                `cfg:"provider" default:"xray" validate:"required"`
	Enabled                     bool                  `cfg:"enabled" default:"false"`
	AddressType                 string                `cfg:"addr_type" default:"local" validate:"required"`
	AddressValue                string                `cfg:"add_value" default:""`
	Sampling                    SamplingConfiguration `cfg:"sampling"`
	StreamingMaxSubsegmentCount int                   `cfg:"streaming_max_subsegment_count" default:"20"`
}

var tracerContainer = struct {
	sync.Mutex
	instance Tracer
}{}

func ProvideTracer(config cfg.Config, logger log.Logger) (Tracer, error) {
	tracerContainer.Lock()
	defer tracerContainer.Unlock()

	if tracerContainer.instance != nil {
		return tracerContainer.instance, nil
	}

	instance, err := NewTracer(config, logger)
	if err != nil {
		return nil, err
	}

	tracerContainer.instance = instance

	return tracerContainer.instance, nil
}

func NewTracer(config cfg.Config, logger log.Logger) (Tracer, error) {
	settings := &TracerSettings{}
	config.UnmarshalKey("tracing", settings)

	if !settings.Enabled {
		return NewNoopTracer(), nil
	}

	if _, ok := providers[settings.Provider]; !ok {
		return nil, fmt.Errorf("no tracing provider found for name %s", settings.Provider)
	}

	provider := providers[settings.Provider]

	return provider(config, logger)
}
