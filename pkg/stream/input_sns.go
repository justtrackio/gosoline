package stream

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/sns"
	"github.com/justtrackio/gosoline/pkg/dx"
	"github.com/justtrackio/gosoline/pkg/log"
)

var _ AcknowledgeableInput = &snsInput{}

type SnsInputTarget struct {
	cfg.AppId
	TopicId    string
	Attributes map[string]string
	ClientName string
}

func (t SnsInputTarget) GetAppId() cfg.AppId {
	return t.AppId
}

func (t SnsInputTarget) GetClientName() string {
	return t.ClientName
}

func (t SnsInputTarget) GetTopicId() string {
	return t.TopicId
}

type snsInput struct {
	*sqsInput
}

func NewSnsInput(ctx context.Context, config cfg.Config, logger log.Logger, settings *SnsInputSettings, targets []SnsInputTarget) (*snsInput, error) {
	autoSubscribe := dx.ShouldAutoCreate(config)

	sqsInput, err := NewSqsInput(ctx, config, logger, &SqsInputSettings{
		AppId:               settings.AppId,
		QueueId:             settings.QueueId,
		MaxNumberOfMessages: settings.MaxNumberOfMessages,
		WaitTime:            settings.WaitTime,
		VisibilityTimeout:   settings.VisibilityTimeout,
		RunnerCount:         settings.RunnerCount,
		RedrivePolicy:       settings.RedrivePolicy,
		ClientName:          settings.ClientName,
		Unmarshaller:        UnmarshallerSns,
	})
	if err != nil {
		return nil, fmt.Errorf("can not create sqsInput: %w", err)
	}

	queueArn := sqsInput.GetQueueArn()

	if autoSubscribe {
		var topic sns.Topic
		var topicName string

		for _, target := range targets {
			if topicName, err = sns.GetTopicName(config, target); err != nil {
				return nil, fmt.Errorf("can not get sns topic name for target %s: %w", target.TopicId, err)
			}

			topicSettings := &sns.TopicSettings{
				TopicName:  topicName,
				ClientName: "default",
			}

			if topic, err = sns.NewTopic(ctx, config, logger, topicSettings); err != nil {
				return nil, fmt.Errorf("can not create topic: %w", err)
			}

			if err = topic.SubscribeSqs(ctx, queueArn, target.Attributes); err != nil {
				return nil, fmt.Errorf("can not subscribe to queue: %w", err)
			}
		}
	}

	return NewSnsInputWithInterfaces(sqsInput), nil
}

func NewSnsInputWithInterfaces(sqsInput *sqsInput) *snsInput {
	return &snsInput{
		sqsInput,
	}
}
