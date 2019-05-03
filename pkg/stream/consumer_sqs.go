package stream

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"time"
)

type SqsConsumerConfig struct {
	TargetFamily      string        `cfg:"consumer_target_family"`
	TargetApplication string        `cfg:"consumer_target_application"`
	TargetQueueId     string        `cfg:"consumer_target_queue_id"`
	IdleTimeout       time.Duration `cfg:"consumer_idle_timeout"`
	WaitTime          int           `cfg:"consumer_wait_time"`
}

type sqsConsumer struct {
	baseConsumer
}

func NewSqsConsumer(callback ConsumerCallback) *sqsConsumer {
	return &sqsConsumer{
		baseConsumer{
			callback: callback,
		},
	}
}

func (c *sqsConsumer) Boot(config cfg.Config, logger mon.Logger) error {
	err := c.baseConsumer.Boot(config, logger)

	if err != nil {
		return err
	}

	cc := &SqsConsumerConfig{}
	config.Bind(cc)

	targetApp := cfg.GetAppIdFromConfig(config)
	targetApp.Family = cc.TargetFamily
	targetApp.Application = cc.TargetApplication

	c.name = fmt.Sprintf("consumer-%v-%v-%v", targetApp.Family, targetApp.Application, cc.TargetQueueId)
	c.input = NewSqsInput(config, logger, SqsInputSettings{
		AppId:    targetApp,
		QueueId:  cc.TargetQueueId,
		WaitTime: int64(cc.WaitTime),
	})

	return nil
}
