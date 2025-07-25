package logging

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/twmb/franz-go/pkg/kgo"
)

const KafkaLoggingChannel = "stream.kafka"

type kafkaLogger struct {
	logger log.Logger
}

func NewKafkaLogger(logger log.Logger) kgo.Logger {
	return kafkaLogger{
		logger: logger.WithChannel(KafkaLoggingChannel),
	}
}

func (l kafkaLogger) Level() kgo.LogLevel {
	return kgo.LogLevelDebug
}

func (l kafkaLogger) Log(level kgo.LogLevel, msg string, keyvals ...any) {
	fields := map[string]any{}

	for i := 0; i < len(keyvals)-1; i += 2 {
		fields[fmt.Sprintf("%v", keyvals[i])] = keyvals[i+1]
	}

	switch level {
	case kgo.LogLevelError:
		l.logger.WithFields(fields).Error(msg)
	case kgo.LogLevelWarn:
		l.logger.WithFields(fields).Warn(msg)
	case kgo.LogLevelInfo:
		l.logger.WithFields(fields).Info(msg)
	case kgo.LogLevelDebug:
		l.logger.WithFields(fields).Debug(msg)
	default:
	}
}
