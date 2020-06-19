package stream

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/refl"
)

type ProducerSettings struct {
	Output      string `cfg:"output"`
	Encoding    string `cfg:"encoding"`
	Compression string `cfg:"compression"`
}

type Producer interface {
	WriteOne(ctx context.Context, model interface{}, attributeSets ...map[string]interface{}) error
	Write(ctx context.Context, models interface{}, attributeSets ...map[string]interface{}) error
}

type producer struct {
	encoder MessageEncoder
	output  Output
}

func NewProducer(config cfg.Config, logger mon.Logger, name string, handlers ...EncodeHandler) *producer {
	key := fmt.Sprintf("stream.producer.%s", name)

	settings := &ProducerSettings{}
	config.UnmarshalKey(key, settings)

	if len(settings.Output) == 0 {
		settings.Output = name
	}

	encodeHandlers := make([]EncodeHandler, 0, len(defaultEncodeHandlers)+len(handlers))
	encodeHandlers = append(encodeHandlers, defaultEncodeHandlers...)
	encodeHandlers = append(encodeHandlers, handlers...)

	encoder := NewMessageEncoder(&MessageEncoderSettings{
		Encoding:       settings.Encoding,
		Compression:    settings.Compression,
		EncodeHandlers: encodeHandlers,
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

func (p *producer) WriteOne(ctx context.Context, model interface{}, attributeSets ...map[string]interface{}) error {
	msg, err := p.encoder.Encode(ctx, model, attributeSets...)

	if err != nil {
		return fmt.Errorf("can not encode model into message: %w", err)
	}

	err = p.output.WriteOne(ctx, msg)

	if err != nil {
		return fmt.Errorf("can not write msg to output: %w", err)
	}

	return nil
}

func (p *producer) Write(ctx context.Context, models interface{}, attributeSets ...map[string]interface{}) error {
	slice, err := refl.InterfaceToInterfaceSlice(models)

	if err != nil {
		return fmt.Errorf("can not cast models interface to slice: %w", err)
	}

	messages := make([]*Message, len(slice))
	for i, model := range slice {
		msg, err := p.encoder.Encode(ctx, model, attributeSets...)

		if err != nil {
			return fmt.Errorf("can not encode model into message: %w", err)
		}

		messages[i] = msg
	}

	err = p.output.Write(ctx, messages)

	if err != nil {
		return fmt.Errorf("can not write messages to output: %w", err)
	}

	return nil
}

func ConfigurableProducerKey(name string) string {
	return fmt.Sprintf("stream.producer.%s", name)
}
