package sns

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/reslife"
)

type LifecycleManager struct {
	service  *Service
	settings *TopicSettings
	topicArn *string
}

func NewLifecycleManager(settings *TopicSettings, topicArn *string) func() (string, reslife.LifeCycleerFactory) {
	return func() (string, reslife.LifeCycleerFactory) {
		id := fmt.Sprintf("sns/%s", settings.TopicName)

		return id, func(ctx context.Context, config cfg.Config, logger log.Logger) (reslife.LifeCycleer, error) {
			var err error
			var service *Service

			if service, err = NewService(ctx, config, logger, settings.ClientName); err != nil {
				return nil, fmt.Errorf("failed to create sns service %w", err)
			}

			return &LifecycleManager{
				service:  service,
				settings: settings,
				topicArn: topicArn,
			}, nil
		}
	}
}

func (l *LifecycleManager) Create(ctx context.Context) error {
	var err error

	if *l.topicArn, err = l.service.CreateTopic(ctx, l.settings.TopicName); err != nil {
		return fmt.Errorf("can not create create topic %s: %w", l.settings.TopicName, err)
	}

	return nil
}

func (l *LifecycleManager) Register(ctx context.Context) (string, any, error) {
	metadata := TopicMetadata{
		AwsClientName: l.settings.ClientName,
		TopicArn:      *l.topicArn,
		TopicName:     l.settings.TopicName,
	}

	return MetadataKeyTopics, metadata, nil
}

func (l *LifecycleManager) Purge(ctx context.Context) error {
	return nil
}
