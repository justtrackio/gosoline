package sqs

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/reslife"
)

const MetadataKeyQueues = "cloud.aws.sqs.queues"

type QueueMetadata struct {
	AwsClientName string `json:"aws_client_name"`
	QueueArn      string `json:"queue_arn"`
	QueueName     string `json:"queue_name"`
	QueueNameFull string `json:"queue_name_full"`
	QueueUrl      string `json:"queue_url"`
}

type lifecycleManager struct {
	service  Service
	settings *Settings
	props    *Properties
}

type LifecycleManager interface {
	reslife.LifeCycleer
	reslife.Creator
	reslife.Initializer
	reslife.Registerer
	reslife.Purger
}

var _ LifecycleManager = (*lifecycleManager)(nil)

func NewLifecycleManager(settings *Settings, props *Properties, optFns ...ClientOption) reslife.LifeCycleerFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (reslife.LifeCycleer, error) {
		var err error
		var svc Service

		if svc, err = NewService(ctx, config, logger, settings, optFns...); err != nil {
			return nil, fmt.Errorf("could not create sqs propertiesResolver: %w", err)
		}

		return &lifecycleManager{
			service:  svc,
			settings: settings,
			props:    props,
		}, nil
	}
}

func (l *lifecycleManager) GetId() string {
	return fmt.Sprintf("sqs/%s", l.settings.QueueName)
}

func (l *lifecycleManager) Create(ctx context.Context) error {
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

func (l *lifecycleManager) Init(ctx context.Context) error {
	var err error
	var props *Properties

	if props, err = l.service.GetPropertiesByName(ctx, l.settings.QueueName); err != nil {
		return fmt.Errorf("could not get sqs queue properties: %w", err)
	}

	l.props.Name = props.Name
	l.props.Url = props.Url
	l.props.Arn = props.Arn

	return nil
}

func (l *lifecycleManager) Register(ctx context.Context) (key string, metadata any, err error) {
	var props *Properties

	if props, err = l.service.GetPropertiesByName(ctx, l.settings.QueueName); err != nil {
		return "", nil, fmt.Errorf("can not get properties: %w", err)
	}

	metadata = QueueMetadata{
		AwsClientName: l.settings.ClientName,
		QueueArn:      props.Arn,
		QueueName:     l.settings.QueueName,
		QueueNameFull: props.Name,
		QueueUrl:      props.Url,
	}

	return MetadataKeyQueues, metadata, nil
}

func (l *lifecycleManager) Purge(ctx context.Context) error {
	if err := l.service.Purge(ctx); err != nil {
		return fmt.Errorf("could not purge sqs queue: %w", err)
	}

	return nil
}
