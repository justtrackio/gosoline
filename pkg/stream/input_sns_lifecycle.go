package stream

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/sns"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/sqs"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/reslife"
)

type LifecycleManager struct {
	propsResolver sqs.PropertiesResolver
	snsService    *sns.Service
	queueName     string
	targets       map[string]SnsInputTarget
}

func NewLifecycleManager(settings *SqsInputSettings, targets []SnsInputTarget) func() (string, reslife.LifeCycleerFactory) {
	return func() (string, reslife.LifeCycleerFactory) {
		id := fmt.Sprintf("sns/%s", settings.QueueId)

		return id, func(ctx context.Context, config cfg.Config, logger log.Logger) (reslife.LifeCycleer, error) {
			var err error
			var queueName, topicName string
			var propsResolver sqs.PropertiesResolver
			var snsService *sns.Service

			if queueName, err = sqs.GetQueueName(config, settings); err != nil {
				return nil, fmt.Errorf("can not get sqs queue name: %w", err)
			}

			if propsResolver, err = sqs.NewPropertiesResolver(ctx, config, logger, settings.ClientName); err != nil {
				return nil, fmt.Errorf("failed to create sns service %w", err)
			}

			if snsService, err = sns.NewService(ctx, config, logger, "default"); err != nil {
				return nil, fmt.Errorf("failed to create sns service %w", err)
			}

			targetMap := map[string]SnsInputTarget{}
			for _, target := range targets {
				if topicName, err = sns.GetTopicName(config, target); err != nil {
					return nil, fmt.Errorf("can not get sns topic name for target %s: %w", target.TopicId, err)
				}

				targetMap[topicName] = target
			}

			return &LifecycleManager{
				propsResolver: propsResolver,
				snsService:    snsService,
				queueName:     queueName,
				targets:       targetMap,
			}, nil
		}
	}
}

func (l *LifecycleManager) Create(ctx context.Context) error {
	var err error
	var props *sqs.Properties
	var topicArn string

	if props, err = l.propsResolver.GetPropertiesByName(ctx, l.queueName); err != nil {
		return fmt.Errorf("can not get sqs properties: %w", err)
	}

	for topicName, target := range l.targets {
		if topicArn, err = l.snsService.CreateTopic(ctx, topicName); err != nil {
			return fmt.Errorf("can not create topic %s: %w", topicName, err)
		}

		if err = l.snsService.SubscribeSqs(ctx, props.Arn, topicArn, target.Attributes); err != nil {
			return fmt.Errorf("can not subscribe to queue: %w", err)
		}
	}

	return nil
}

func (l *LifecycleManager) Register(ctx context.Context) (string, any, error) {
	return "", nil, nil
}

func (l *LifecycleManager) Purge(ctx context.Context) error {
	return nil
}
