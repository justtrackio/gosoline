package stream

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/tracing"
)

type outputTracer struct {
	tracer tracing.Tracer
	base   Output
	name   string
}

func NewOutputTracer(config cfg.Config, logger mon.Logger, base Output, name string) *outputTracer {
	key := ConfigurableOutputKey(name)

	settings := &BaseOutputSettings{}
	config.UnmarshalKey(key, settings)

	tracer := tracing.NewNoopTracer()

	if settings.Tracing.Enabled {
		tracer = tracing.ProviderTracer(config, logger)
	}

	return NewOutputTracerWithInterfaces(tracer, base, name)
}

func NewOutputTracerWithInterfaces(tracer tracing.Tracer, base Output, name string) *outputTracer {
	return &outputTracer{
		tracer: tracer,
		base:   base,
		name:   name,
	}
}

func (o outputTracer) WriteOne(ctx context.Context, msg WritableMessage) error {
	spanName := fmt.Sprintf("stream-output-%s", o.name)

	ctx, trans := o.tracer.StartSubSpan(ctx, spanName)
	defer trans.Finish()

	return o.base.WriteOne(ctx, msg)
}

func (o outputTracer) Write(ctx context.Context, batch []WritableMessage) error {
	spanName := fmt.Sprintf("stream-output-%s", o.name)

	ctx, trans := o.tracer.StartSubSpan(ctx, spanName)
	defer trans.Finish()

	return o.base.Write(ctx, batch)
}
