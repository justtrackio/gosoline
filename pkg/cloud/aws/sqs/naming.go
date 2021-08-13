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

type QueueNameSettingsAware interface {
	GetAppid() cfg.AppId
	GetQueueId() string
	IsFifoEnabled() bool
}

type QueueNameSettings struct {
	AppId       cfg.AppId
	QueueId     string
	FifoEnabled bool
}

func (q QueueNameSettings) GetAppid() cfg.AppId {
	return q.AppId
}

func (q QueueNameSettings) GetQueueId() string {
	return q.QueueId
}

func (q QueueNameSettings) IsFifoEnabled() bool {
	return q.FifoEnabled
}

func GetQueueName(settings QueueNameSettingsAware) string {
	name := namingStrategy(settings.GetAppid(), settings.GetQueueId())

	if settings.IsFifoEnabled() {
		name = name + fifoSuffix
	}

	return name
}
