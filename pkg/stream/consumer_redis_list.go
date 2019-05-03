package stream

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"time"
)

type RedisListConsumerConfig struct {
	TargetFamily      string `cfg:"consumer_target_family"`
	TargetApplication string `cfg:"consumer_target_application"`
	TargetKey         string `cfg:"consumer_target_key"`
	WaitTime          int    `cfg:"consumer_wait_time"`
}

type redisListConsumer struct {
	baseConsumer
}

func NewRedisListConsumer(callback ConsumerCallback) *redisListConsumer {
	return &redisListConsumer{
		baseConsumer{
			callback: callback,
		},
	}
}

func (c *redisListConsumer) Boot(config cfg.Config, logger mon.Logger) error {
	err := c.baseConsumer.Boot(config, logger)

	if err != nil {
		return err
	}

	cc := &RedisListConsumerConfig{}
	config.Bind(cc)

	targetApp := cfg.GetAppIdFromConfig(config)
	targetApp.Family = cc.TargetFamily
	targetApp.Application = cc.TargetApplication

	c.name = fmt.Sprintf("consumer-%v-%v-%v", targetApp.Family, targetApp.Application, cc.TargetKey)
	c.input = NewRedisListInput(config, logger, &RedisListInputSettings{
		AppId:      targetApp,
		ServerName: "default",
		Key:        cc.TargetKey,
		WaitTime:   time.Duration(cc.WaitTime) * time.Second,
	})

	return nil
}
