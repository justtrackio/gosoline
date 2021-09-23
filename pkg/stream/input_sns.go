package stream

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/sns"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/sqs"
	"github.com/justtrackio/gosoline/pkg/dx"
	"github.com/justtrackio/gosoline/pkg/log"
)

var _ AcknowledgeableInput = &snsInput{}

type SnsInputSettings struct {
	cfg.AppId
	QueueId             string            `cfg:"queue_id"`
	MaxNumberOfMessages int32             `cfg:"max_number_of_messages" default:"10" validate:"min=1,max=10"`
	WaitTime            int32             `cfg:"wait_time"`
	RedrivePolicy       sqs.RedrivePolicy `cfg:"redrive_policy"`
	VisibilityTimeout   int               `cfg:"visibility_timeout"`
	RunnerCount         int               `cfg:"runner_count"`
	ClientName          string            `cfg:"client_name"`
}

func (s SnsInputSettings) GetAppid() cfg.AppId {
	return s.AppId
}

func (s SnsInputSettings) GetQueueId() string {
	return s.QueueId
}

func (s SnsInputSettings) IsFifoEnabled() bool {
	return false
}

type SnsInputTarget struct {
	cfg.AppId
	TopicId    string
	Attributes map[string]interface{}
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

		for _, target := range targets {
			topicName := sns.GetTopicName(target.AppId, target.TopicId)
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
