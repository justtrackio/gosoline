package stream

import (
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/sqs"
)

type BaseOutputConfiguration struct {
	Tracing BaseOutputConfigurationTracing `cfg:"tracing"`
}

func (b *BaseOutputConfiguration) SetTracing(enabled bool) {
	b.Tracing.Enabled = enabled
}

type BaseOutputConfigurationTracing struct {
	Enabled bool `cfg:"enabled" default:"true"`
}

type FileOutputSettings struct {
	Filename string         `cfg:"filename"`
	Mode     FileOutputMode `cfg:"mode"     default:"append"`
}

type KinesisOutputSettings struct {
	cfg.AppId
	ClientName string
	StreamName string
}

func (s KinesisOutputSettings) GetAppId() cfg.AppId {
	return s.AppId
}

func (s KinesisOutputSettings) GetClientName() string {
	return s.ClientName
}

func (s KinesisOutputSettings) GetStreamName() string {
	return s.StreamName
}

const (
	metricNameRedisListOutputWrites = "StreamRedisListOutputWrites"
)

type RedisListOutputSettings struct {
	cfg.AppId
	ServerName string
	Key        string
	BatchSize  int
}

type SnsOutputSettings struct {
	cfg.AppId
	TopicId    string
	ClientName string
}

func (s SnsOutputSettings) GetAppId() cfg.AppId {
	return s.AppId
}

func (s SnsOutputSettings) GetClientName() string {
	return s.ClientName
}

func (s SnsOutputSettings) GetTopicId() string {
	return s.TopicId
}

type SqsOutputSettings struct {
	cfg.AppId
	ClientName        string
	Fifo              sqs.FifoSettings
	QueueId           string
	RedrivePolicy     sqs.RedrivePolicy
	VisibilityTimeout int
}

func (s SqsOutputSettings) GetAppId() cfg.AppId {
	return s.AppId
}

func (s SqsOutputSettings) GetClientName() string {
	return s.ClientName
}

func (s SqsOutputSettings) IsFifoEnabled() bool {
	return s.Fifo.Enabled
}

func (s SqsOutputSettings) GetQueueId() string {
	return s.QueueId
}
