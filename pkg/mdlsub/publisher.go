package mdlsub

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/stream"
)

const (
	ConfigKeyMdlSubPublishers = "mdlsub.publishers"
	TypeCreate                = "create"
	TypeUpdate                = "update"
	TypeDelete                = "delete"
)

type PublisherSettings struct {
	mdl.ModelId
	Producer   string `cfg:"producer" validate:"required_without=OutputType"`
	OutputType string `cfg:"output_type" validate:"required_without=Producer"`
	Shared     bool   `cfg:"shared"`
}

//go:generate mockery -name Publisher
type Publisher interface {
	Publish(ctx context.Context, typ string, version int, value interface{}, customAttributes ...map[string]interface{}) error
}

type publisher struct {
	logger   mon.Logger
	producer stream.Producer
	settings *PublisherSettings
}

func NewPublisher(config cfg.Config, logger mon.Logger, name string) *publisher {
	settings := readPublisherSetting(config, name)

	return NewPublisherWithSettings(config, logger, settings)
}

func NewPublisherWithSettings(config cfg.Config, logger mon.Logger, settings *PublisherSettings) *publisher {
	producer := stream.NewProducer(config, logger, settings.Producer)

	return NewPublisherWithInterfaces(logger, producer, settings)
}

func NewPublisherWithInterfaces(logger mon.Logger, producer stream.Producer, settings *PublisherSettings) *publisher {
	return &publisher{
		logger:   logger,
		producer: producer,
		settings: settings,
	}
}

func (p *publisher) Publish(ctx context.Context, typ string, version int, value interface{}, customAttributes ...map[string]interface{}) error {
	attributes := []map[string]interface{}{
		CreateMessageAttributes(p.settings.ModelId, typ, version),
	}
	attributes = append(attributes, customAttributes...)

	if err := p.producer.WriteOne(ctx, value, attributes...); err != nil {
		return fmt.Errorf("can not publish %s with publisher %s: %w", p.settings.ModelId.String(), p.settings.Name, err)
	}

	return nil
}

func CreateMessageAttributes(modelId mdl.ModelId, typ string, version int) map[string]interface{} {
	return map[string]interface{}{
		"type":    typ,
		"version": version,
		"modelId": modelId.String(),
	}
}
