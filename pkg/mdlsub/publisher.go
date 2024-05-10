package mdlsub

import (
	"context"
	"fmt"
	"strconv"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/stream"
)

const (
	AttributeModelId          = "modelId"
	AttributeType             = "type"
	AttributeVersion          = "version"
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

//go:generate go run github.com/vektra/mockery/v2 --name Publisher
type Publisher interface {
	PublishBatch(ctx context.Context, typ string, version int, values []any, customAttributes ...map[string]string) error
	Publish(ctx context.Context, typ string, version int, value any, customAttributes ...map[string]string) error
}

type publisher struct {
	logger   log.Logger
	producer stream.Producer
	settings *PublisherSettings
}

func NewPublisher(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Publisher, error) {
	settings, err := readPublisherSetting(config, name)
	if err != nil {
		return nil, fmt.Errorf("can not read publisher settings for %s: %w", name, err)
	}

	return NewPublisherWithSettings(ctx, config, logger, settings)
}

func NewPublisherWithSettings(ctx context.Context, config cfg.Config, logger log.Logger, settings *PublisherSettings) (Publisher, error) {
	var err error
	var producer stream.Producer

	if producer, err = stream.NewProducer(ctx, config, logger, settings.Producer); err != nil {
		return nil, fmt.Errorf("can not create producer %s: %w", settings.Producer, err)
	}

	return NewPublisherWithInterfaces(logger, producer, settings), nil
}

func NewPublisherWithInterfaces(logger log.Logger, producer stream.Producer, settings *PublisherSettings) Publisher {
	return &publisher{
		logger:   logger,
		producer: producer,
		settings: settings,
	}
}

func (p *publisher) PublishBatch(ctx context.Context, typ string, version int, values []any, customAttributes ...map[string]string) error {
	attributes := []map[string]string{
		CreateMessageAttributes(p.settings.ModelId, typ, version),
	}
	attributes = append(attributes, customAttributes...)

	if err := p.producer.Write(ctx, values, attributes...); err != nil {
		return fmt.Errorf("can not publish %s with publisher %s: %w", p.settings.String(), p.settings.Name, err)
	}

	return nil
}

func (p *publisher) Publish(ctx context.Context, typ string, version int, value any, customAttributes ...map[string]string) error {
	attributes := []map[string]string{
		CreateMessageAttributes(p.settings.ModelId, typ, version),
	}
	attributes = append(attributes, customAttributes...)

	if err := p.producer.WriteOne(ctx, value, attributes...); err != nil {
		return fmt.Errorf("can not publish %s with publisher %s: %w", p.settings.String(), p.settings.Name, err)
	}

	return nil
}

func CreateMessageAttributes(modelId mdl.ModelId, typ string, version int) map[string]string {
	return map[string]string{
		AttributeType:    typ,
		AttributeVersion: strconv.Itoa(version),
		AttributeModelId: modelId.String(),
	}
}
