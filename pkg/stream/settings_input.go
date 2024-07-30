package stream

import (
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/sqs"
)

type FileSettings struct {
	Filename string `cfg:"filename"`
	Blocking bool   `cfg:"blocking"`
}

type InMemorySettings struct {
	Size int `cfg:"size" default:"1"`
}

type RedisListInputSettings struct {
	cfg.AppId
	ServerName string
	Key        string
	WaitTime   time.Duration
}

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

func (s SnsInputSettings) GetAppId() cfg.AppId {
	return s.AppId
}

func (s SnsInputSettings) GetClientName() string {
	return s.ClientName
}

func (s SnsInputSettings) GetQueueId() string {
	return s.QueueId
}

func (s SnsInputSettings) IsFifoEnabled() bool {
	return false
}

type SqsInputSettings struct {
	cfg.AppId
	QueueId             string            `cfg:"queue_id"`
	MaxNumberOfMessages int32             `cfg:"max_number_of_messages" default:"10"  validate:"min=1,max=10"`
	WaitTime            int32             `cfg:"wait_time"`
	VisibilityTimeout   int               `cfg:"visibility_timeout"`
	RunnerCount         int               `cfg:"runner_count"`
	Fifo                sqs.FifoSettings  `cfg:"fifo"`
	RedrivePolicy       sqs.RedrivePolicy `cfg:"redrive_policy"`
	ClientName          string            `cfg:"client_name"`
	Unmarshaller        string            `cfg:"unmarshaller"           default:"msg"`
}

func (s SqsInputSettings) GetAppId() cfg.AppId {
	return s.AppId
}

func (s SqsInputSettings) GetClientName() string {
	return s.ClientName
}

func (s SqsInputSettings) GetQueueId() string {
	return s.QueueId
}

func (s SqsInputSettings) IsFifoEnabled() bool {
	return s.Fifo.Enabled
}
