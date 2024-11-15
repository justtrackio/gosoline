package tracing

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/justtrackio/gosoline/pkg/appctx"
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
	Provider                    string                `cfg:"provider" default:"local" validate:"required"`
	AddressType                 string                `cfg:"addr_type" default:"local" validate:"required"`
	AddressValue                string                `cfg:"add_value" default:""`
	Sampling                    SamplingConfiguration `cfg:"sampling"`
	StreamingMaxSubsegmentCount int                   `cfg:"streaming_max_subsegment_count" default:"20"`
}

type tracerKey struct{}

func ProvideTracer(ctx context.Context, config cfg.Config, logger log.Logger) (Tracer, error) {
	return appctx.Provide(ctx, tracerKey{}, func() (Tracer, error) {
		return newTracer(config, logger)
	})
}

func newTracer(config cfg.Config, logger log.Logger) (Tracer, error) {
	var provider Provider
	var ok bool

	settings := &TracerSettings{}
	config.UnmarshalKey("tracing", settings)

	if provider, ok = providers[settings.Provider]; !ok {
		return nil, fmt.Errorf(
			"no tracing provider found for name %s, available providers: %s",
			settings.Provider,
			strings.Join(slices.Collect(maps.Keys(providers)), ", "),
		)
	}

	return provider(config, logger)
}
