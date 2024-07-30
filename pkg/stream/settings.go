package stream

import (
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

const (
	ConfigKeyStream = "stream"
)

type ConsumerSettings struct {
	Input       string                `cfg:"input"        default:"consumer"         validate:"required"`
	RunnerCount int                   `cfg:"runner_count" default:"1"                validate:"min=1"`
	Encoding    EncodingType          `cfg:"encoding"     default:"application/json"`
	IdleTimeout time.Duration         `cfg:"idle_timeout" default:"10s"`
	Retry       ConsumerRetrySettings `cfg:"retry"`
}

type ConsumerRetrySettings struct {
	Enabled bool   `cfg:"enabled"`
	Type    string `cfg:"type"    default:"sqs"`
}

type BatchConsumerSettings struct {
	IdleTimeout time.Duration `cfg:"idle_timeout" default:"10s"`
	BatchSize   int           `cfg:"batch_size"   default:"1"`
}

type MessageEncoderSettings struct {
	Encoding       EncodingType
	Compression    CompressionType
	EncodeHandlers []EncodeHandler
}

type ProducerSettings struct {
	Output      string                 `cfg:"output"`
	Encoding    EncodingType           `cfg:"encoding"`
	Compression CompressionType        `cfg:"compression" default:"none"`
	Daemon      ProducerDaemonSettings `cfg:"daemon"`
}

type RetryHandlerSqsSettings struct {
	cfg.AppId
	RetryHandlerSettings
	ClientName          string `cfg:"client_name"            default:"default"`
	MaxNumberOfMessages int32  `cfg:"max_number_of_messages" default:"10"      validate:"min=1,max=10"`
	WaitTime            int32  `cfg:"wait_time"              default:"10"`
	RunnerCount         int    `cfg:"runner_count"           default:"1"`
	QueueId             string `cfg:"queue_id"`
}
