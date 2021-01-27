package sqs

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
)

const fifoSuffix = ".fifo"

type NamingFactory func(appId cfg.AppId, queueId string) string

var namingStrategy = func(appId cfg.AppId, queueId string) string {
	return fmt.Sprintf("%v-%v-%v-%v-%v", appId.Project, appId.Environment, appId.Family, appId.Application, queueId)
}

func WithNamingStrategy(strategy NamingFactory) {
	namingStrategy = strategy
}

var deadLetterNamingStrategy = func(appId cfg.AppId, queueId string) string {
	return fmt.Sprintf("%v-%v-%v-%v-%v-dead", appId.Project, appId.Environment, appId.Family, appId.Application, queueId)
}

func WithDeadLetterNamingStrategy(strategy NamingFactory) {
	deadLetterNamingStrategy = strategy
}

type QueueNameSettings interface {
	GetAppid() cfg.AppId
	GetQueueId() string
	IsFifoEnabled() bool
}

func QueueName(settings QueueNameSettings) string {
	name := namingStrategy(settings.GetAppid(), settings.GetQueueId())

	if settings.IsFifoEnabled() {
		name = name + fifoSuffix
	}

	return name
}
