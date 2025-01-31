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

type lifecycleManager struct {
	propsResolver sqs.PropertiesResolver
	snsService    *sns.Service
	queueName     string
	targets       map[string]SnsInputTarget
}

type LifecycleManager interface {
	reslife.LifeCycleer
	reslife.Creator
}

var _ LifecycleManager = (*lifecycleManager)(nil)

func NewLifecycleManager(settings *SqsInputSettings, targets []SnsInputTarget) reslife.LifeCycleerFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (reslife.LifeCycleer, error) {
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

		return &lifecycleManager{
			propsResolver: propsResolver,
			snsService:    snsService,
			queueName:     queueName,
			targets:       targetMap,
		}, nil
	}
}

func (l *lifecycleManager) GetId() string {
	return fmt.Sprintf("sns/%s", l.queueName)
}

func (l *lifecycleManager) Create(ctx context.Context) error {
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
