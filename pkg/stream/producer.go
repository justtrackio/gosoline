package stream

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/refl"
)

const MetadataKeyProducers = "stream.producers"

type ProducerMetadata struct {
	Name          string `json:"name"`
	DaemonEnabled bool   `json:"daemon_enabled"`
}

type ProducerSettings struct {
	Output      string                 `cfg:"output"`
	Encoding    EncodingType           `cfg:"encoding"`
	Compression CompressionType        `cfg:"compression" default:"none"`
	Daemon      ProducerDaemonSettings `cfg:"daemon"`
}

//go:generate mockery --name Producer
type Producer interface {
	WriteOne(ctx context.Context, model interface{}, attributeSets ...map[string]string) error
	Write(ctx context.Context, models interface{}, attributeSets ...map[string]string) error
}

type producer struct {
	encoder MessageEncoder
	output  Output
}

func NewProducer(ctx context.Context, config cfg.Config, logger log.Logger, name string, handlers ...EncodeHandler) (*producer, error) {
	settings := readProducerSettings(config, name)

	var err error
	var output Output

	if !settings.Daemon.Enabled {
		if output, err = NewConfigurableOutput(ctx, config, logger, settings.Output); err != nil {
			return nil, fmt.Errorf("can not create output %s: %w", settings.Output, err)
		}
	} else {
		// the producer daemon will take care of compression for the whole batch, so we don't need to compress individual messages
		settings.Compression = CompressionNone
		if output, err = ProvideProducerDaemon(ctx, config, logger, name); err != nil {
			return nil, fmt.Errorf("can not create producer daemon %s: %w", name, err)
		}
	}

	encodeHandlers := make([]EncodeHandler, 0, len(defaultEncodeHandlers)+len(handlers))
	encodeHandlers = append(encodeHandlers, defaultEncodeHandlers...)
	encodeHandlers = append(encodeHandlers, handlers...)

	encoder := NewMessageEncoder(&MessageEncoderSettings{
		Encoding:       settings.Encoding,
		Compression:    settings.Compression,
		EncodeHandlers: encodeHandlers,
	})

	metadata := ProducerMetadata{
		Name:          name,
		DaemonEnabled: settings.Daemon.Enabled,
	}

	if err = appctx.MetadataAppend(ctx, MetadataKeyProducers, metadata); err != nil {
		return nil, fmt.Errorf("can not access the appctx metadata: %w", err)
	}

	return NewProducerWithInterfaces(encoder, output), nil
}

func NewProducerWithInterfaces(encoder MessageEncoder, output Output) *producer {
	return &producer{
		encoder: encoder,
		output:  output,
	}
}

func (p *producer) WriteOne(ctx context.Context, model interface{}, attributeSets ...map[string]string) error {
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

func (p *producer) Write(ctx context.Context, models interface{}, attributeSets ...map[string]string) error {
	slice, err := refl.InterfaceToInterfaceSlice(models)
	if err != nil {
		return fmt.Errorf("can not cast models interface to slice: %w", err)
	}

	messages := make([]WritableMessage, len(slice))
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

func readProducerSettings(config cfg.Config, name string) *ProducerSettings {
	key := ConfigurableProducerKey(name)

	settings := &ProducerSettings{}
	config.UnmarshalKey(key, settings)

	if settings.Encoding == "" {
		settings.Encoding = defaultMessageBodyEncoding
	}

	if settings.Output == "" {
		settings.Output = name
	}

	return settings
}

func readAllProducerDaemonSettings(config cfg.Config) map[string]*ProducerSettings {
	producerSettings := make(map[string]*ProducerSettings)
	producerMap := config.GetStringMap("stream.producer", map[string]interface{}{})

	for name := range producerMap {
		settings := readProducerSettings(config, name)

		if !settings.Daemon.Enabled {
			continue
		}

		producerSettings[name] = settings
	}

	return producerSettings
}
