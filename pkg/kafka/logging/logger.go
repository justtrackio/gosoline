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

func NewKafkaLogger(logger log.Logger) *KafkaLogger {
	return &KafkaLogger{Logger: logger.WithChannel(KafkaLoggingChannel)}
}

func (l *KafkaLogger) DebugLogger() LoggerWrapper {
	return l.Debug
}

func (l *KafkaLogger) ErrorLogger() LoggerWrapper {
	return l.Error
}
