package sqs

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/dx"
	"github.com/justtrackio/gosoline/pkg/log"
)

type LifecycleManager struct {
	service  Service
	settings *Settings
	props    *Properties
}

func NewLifecycleManager(settings *Settings, props *Properties, optFns ...ClientOption) func() (string, dx.LifeCycleerFactory) {
	return func() (string, dx.LifeCycleerFactory) {
		id := fmt.Sprintf("sqs/%s", settings.QueueName)

		return id, func(ctx context.Context, config cfg.Config, logger log.Logger) (dx.LifeCycleer, error) {
			var err error
			var svc Service

			if svc, err = NewService(ctx, config, logger, settings, optFns...); err != nil {
				return nil, fmt.Errorf("could not create ddb service: %w", err)
			}

			return &LifecycleManager{
				service:  svc,
				settings: settings,
				props:    props,
			}, nil
		}
	}
}

func (l *LifecycleManager) Create(ctx context.Context) error {
	var err error
	var props *Properties

	if props, err = l.service.CreateQueue(ctx); err != nil {
		return fmt.Errorf("could not create sqs queue %s: %w", "", err)
	}

	l.props.Name = props.Name
	l.props.Url = props.Url
	l.props.Arn = props.Arn

	return nil
}

func (l *LifecycleManager) Register(ctx context.Context) (string, any, error) {
	var err error
	var props *Properties

	if props, err = l.service.GetPropertiesByName(ctx, l.settings.QueueName); err != nil {
		return "", nil, fmt.Errorf("can not get properties: %w", err)
	}

	metadata := QueueMetadata{
		AwsClientName: l.settings.ClientName,
		QueueArn:      props.Arn,
		QueueName:     l.settings.QueueName,
		QueueNameFull: props.Name,
		QueueUrl:      props.Url,
	}

	return MetadataKeyQueues, metadata, nil
}

func (l *LifecycleManager) Purge(ctx context.Context) error {
	return nil
}
