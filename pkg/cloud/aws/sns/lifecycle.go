package sns

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/reslife"
)

type lifecycleManager struct {
	service  *Service
	settings *TopicSettings
	topicArn *string
}

type LifecycleManager interface {
	reslife.LifeCycleer
	reslife.Creator
	reslife.Initializer
	reslife.Registerer
}

var _ LifecycleManager = (*lifecycleManager)(nil)

func NewLifecycleManager(settings *TopicSettings, topicArn *string) reslife.LifeCycleerFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (reslife.LifeCycleer, error) {
		var err error
		var service *Service

		if service, err = NewService(ctx, config, logger, settings.ClientName); err != nil {
			return nil, fmt.Errorf("failed to create sns service %w", err)
		}

		return &lifecycleManager{
			service:  service,
			settings: settings,
			topicArn: topicArn,
		}, nil
	}
}

func (l *lifecycleManager) GetId() string {
	return fmt.Sprintf("sns/%s", l.settings.TopicName)
}

func (l *lifecycleManager) Create(ctx context.Context) error {
	var err error

	if *l.topicArn, err = l.service.CreateTopic(ctx, l.settings.TopicName); err != nil {
		return fmt.Errorf("can not create topic %s: %w", l.settings.TopicName, err)
	}

	return nil
}

func (l *lifecycleManager) Init(ctx context.Context) error {
	return l.Create(ctx)
}

func (l *lifecycleManager) Register(ctx context.Context) (key string, metadata any, err error) {
	metadata = TopicMetadata{
		AwsClientName: l.settings.ClientName,
		TopicArn:      *l.topicArn,
		TopicName:     l.settings.TopicName,
	}

	return MetadataKeyTopics, metadata, nil
}
