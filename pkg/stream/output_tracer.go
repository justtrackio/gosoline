package stream

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/tracing"
)

type outputTracer struct {
	tracer tracing.Tracer
	base   Output
	name   string
}

func NewOutputTracer(config cfg.Config, logger log.Logger, base Output, name string) (*outputTracer, error) {
	key := ConfigurableOutputKey(name)

	settings := &BaseOutputConfiguration{}
	config.UnmarshalKey(key, settings)

	var err error
	tracer := tracing.NewNoopTracer()

	if settings.Tracing.Enabled {
		if tracer, err = tracing.ProvideTracer(config, logger); err != nil {
			return nil, fmt.Errorf("can not create tracer: %w", err)
		}
	}

	return NewOutputTracerWithInterfaces(tracer, base, name), nil
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
