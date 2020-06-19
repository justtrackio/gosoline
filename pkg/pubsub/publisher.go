package pubsub

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/stream"
)

const (
	TypeCreate = "create"
)

type PublisherSettings struct {
	Producer    string `cfg:"producer" validate:"required_without=OutputType"`
	OutputType  string `cfg:"output_type" validate:"required_without=Producer"`
	Shared      bool   `cfg:"shared"`
	Project     string `cfg:"project"`
	Family      string `cfg:"family"`
	Application string `cfg:"application"`
	Name        string `cfg:"name" validate:"required"`
}

type Publisher interface {
	Publish(ctx context.Context, typ string, version int, value interface{}) error
}

type publisher struct {
	logger   mon.Logger
	producer stream.Producer
	settings *PublisherSettings
}

func NewPublisherFromConfig(config cfg.Config, logger mon.Logger, name string) *publisher {
	var settings *PublisherSettings
	var allSettings = readPublisherSettings(config)

	for _, s := range allSettings {
		if s.Name == name {
			settings = s
		}
	}

	if settings == nil {
		err := fmt.Errorf("there is no publisher configured with name %s", name)
		logger.Fatalf(err, err.Error())
		return nil
	}

	producer := stream.NewProducer(config, logger, settings.Producer)

	return NewPublisher(logger, producer, settings)
}

func NewPublisher(logger mon.Logger, producer stream.Producer, settings *PublisherSettings) *publisher {
	return &publisher{
		logger:   logger,
		producer: producer,
		settings: settings,
	}
}

func (p *publisher) Publish(ctx context.Context, typ string, version int, value interface{}) error {
	modelId := fmt.Sprintf("%s.%s.%s.%s", p.settings.Project, p.settings.Family, p.settings.Application, p.settings.Name)

	attributes := map[string]interface{}{
		"type":    typ,
		"version": version,
		"modelId": modelId,
	}

	if err := p.producer.WriteOne(ctx, value, attributes); err != nil {
		return fmt.Errorf("can not publish %s with publisher %s: %w", modelId, p.settings.Name, err)
	}

	return nil
}
