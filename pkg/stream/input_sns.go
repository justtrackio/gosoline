package stream

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/sns"
	"github.com/applike/gosoline/pkg/sqs"
)

type SnsInputSettings struct {
	cfg.AppId
	QueueId             string               `cfg:"queue_id"`
	MaxNumberOfMessages int64                `cfg:"max_number_of_messages" default:"10" validate:"min=1,max=10"`
	WaitTime            int64                `cfg:"wait_time"`
	RedrivePolicy       sqs.RedrivePolicy    `cfg:"redrive_policy"`
	VisibilityTimeout   int                  `cfg:"visibility_timeout"`
	RunnerCount         int                  `cfg:"runner_count"`
	Client              cloud.ClientSettings `cfg:"client"`
	Backoff             exec.BackoffSettings `cfg:"backoff"`
}

type SnsInputTarget struct {
	cfg.AppId
	TopicId    string
	Attributes map[string]interface{}
}

type snsInput struct {
	*sqsInput
}

func NewSnsInput(config cfg.Config, logger mon.Logger, settings SnsInputSettings, targets []SnsInputTarget) *snsInput {
	autoSubscribe := config.GetBool("aws_sns_autoSubscribe")

	sqsInput := NewSqsInput(config, logger, SqsInputSettings{
		AppId:               settings.AppId,
		QueueId:             settings.QueueId,
		MaxNumberOfMessages: settings.MaxNumberOfMessages,
		WaitTime:            settings.WaitTime,
		VisibilityTimeout:   settings.VisibilityTimeout,
		RunnerCount:         settings.RunnerCount,
		RedrivePolicy:       settings.RedrivePolicy,
		Client:              settings.Client,
		Backoff:             settings.Backoff,
		Unmarshaller:        UnmarshallerSns,
	})

	queueArn := sqsInput.GetQueueArn()

	if autoSubscribe {
		for _, target := range targets {
			topic := sns.NewTopic(config, logger, &sns.Settings{
				AppId:   target.AppId,
				TopicId: target.TopicId,
				Client:  settings.Client,
				Backoff: settings.Backoff,
			})

			err := topic.SubscribeSqs(queueArn, target.Attributes)

			if err != nil {
				panic(err)
			}
		}
	}

	return NewSnsInputWithInterfaces(sqsInput)
}

func NewSnsInputWithInterfaces(sqsInput *sqsInput) *snsInput {
	return &snsInput{
		sqsInput,
	}
}
