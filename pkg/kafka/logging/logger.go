package logging

import (
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	KafkaLoggingChannel = "stream.kafka"
)

type KafkaLogger struct {
	log.Logger
}

func NewKafkaLogger(logger log.Logger) KafkaLogger {
	return KafkaLogger{Logger: logger.WithChannel(KafkaLoggingChannel)}
}

func (logger KafkaLogger) DebugLogger() DebugLoggerWrapper {
	return DebugLoggerWrapper{logger}
}

func (logger KafkaLogger) ErrorLogger() ErrorLoggerWrapper {
	return ErrorLoggerWrapper{logger}
}
