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

var _ SchemaRegistryAwareOutput = &outputTracer{}

func NewOutputTracer(ctx context.Context, config cfg.Config, logger log.Logger, base Output, name string) (*outputTracer, error) {
	key := ConfigurableOutputKey(name)

	settings := &BaseOutputConfiguration{}
	if err := config.UnmarshalKey(key, settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal output tracer settings for key %q in NewOutputTracer: %w", key, err)
	}

	var err error
	tracer := tracing.NewLocalTracer()

	if settings.Tracing.Enabled {
		if tracer, err = tracing.ProvideTracer(ctx, config, logger); err != nil {
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

func (o outputTracer) InitSchemaRegistry(ctx context.Context, settings SchemaSettingsWithEncoding) (MessageBodyEncoder, error) {
	if schemaRegistryAwareOutput, ok := o.base.(SchemaRegistryAwareOutput); ok {
		return schemaRegistryAwareOutput.InitSchemaRegistry(ctx, settings)
	}

	return nil, fmt.Errorf("output does not support a schema registry")
}
