package stream

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/sns"
)

type SnsInputSettings struct {
	cfg.AppId
	AutoSubscribe bool
	QueueId       string
	WaitTime      int64
}

type SnsInputTarget struct {
	cfg.AppId
	TopicId string
}

type snsInput struct {
	logger   mon.Logger
	sqsInput *sqsInput
	settings SnsInputSettings
}

func NewSnsInput(config cfg.Config, logger mon.Logger, s SnsInputSettings, targets []SnsInputTarget) Input {
	s.PadFromConfig(config)
	s.AutoSubscribe = config.GetBool("aws_sns_autoSubscribe")

	sqsInput := NewSqsInput(config, logger, SqsInputSettings{
		AppId:    s.AppId,
		QueueId:  s.QueueId,
		WaitTime: s.WaitTime,
	})
	sqsInput.SetUnmarshaler(SnsUnmarshaler)

	queueArn := sqsInput.GetQueueArn()

	if s.AutoSubscribe {
		for _, t := range targets {
			t.PadFromConfig(config)

			topic := sns.NewTopic(config, logger, sns.Settings{
				AppId:   t.AppId,
				TopicId: t.TopicId,
			})

			err := topic.SubscribeSqs(queueArn)

			if err != nil {
				panic(err)
			}
		}
	}

	return NewSnsInputWithInterfaces(logger, sqsInput, s)
}

func NewSnsInputWithInterfaces(logger mon.Logger, sqsInput *sqsInput, s SnsInputSettings) Input {
	return &snsInput{
		logger:   logger,
		sqsInput: sqsInput,
		settings: s,
	}
}

func (i *snsInput) Data() chan *Message {
	return i.sqsInput.Data()
}

func (i *snsInput) Run() error {
	return i.sqsInput.Run()
}

func (i *snsInput) Stop() {
	i.sqsInput.Stop()
}
