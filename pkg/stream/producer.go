package stream

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
)

type ProducerSettings struct {
	Output   string `cfg:"output" default:"producer" validate:"required"`
	Encoding string `cfg:"encoding"`
}

type Producer interface {
	WriteOne(ctx context.Context, model interface{}) error
}

type producer struct {
	encoder MessageEncoder
	output  Output
}

func NewProducer(config cfg.Config, logger mon.Logger, name string) *producer {
	key := fmt.Sprintf("stream.producer.%s", name)

	settings := &ProducerSettings{}
	config.UnmarshalKey(key, settings)

	encoder := NewMessageEncoder(&MessageEncoderSettings{
		Encoding: settings.Encoding,
	})
	output := NewConfigurableOutput(config, logger, settings.Output)

	return NewProducerWithInterfaces(encoder, output)
}

func NewProducerWithInterfaces(encoder MessageEncoder, output Output) *producer {
	return &producer{
		encoder: encoder,
		output:  output,
	}
}

func (p *producer) WriteOne(ctx context.Context, model interface{}) error {
	msg, err := p.encoder.Encode(ctx, model)

	if err != nil {
		return fmt.Errorf("can not encode model into message: %w", err)
	}

	err = p.output.WriteOne(ctx, msg)

	if err != nil {
		return fmt.Errorf("can not write msg to output: %w", err)
	}

	return nil
}
