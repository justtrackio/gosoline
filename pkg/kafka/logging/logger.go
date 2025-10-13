package logging

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/twmb/franz-go/pkg/kgo"
)

const KafkaLoggingChannel = "stream.kafka"

type kafkaLogger struct {
	ctx    context.Context
	logger log.Logger
}

func NewKafkaLogger(ctx context.Context, logger log.Logger) kgo.Logger {
	return kafkaLogger{
		ctx:    ctx,
		logger: logger.WithChannel(KafkaLoggingChannel),
	}
}

func (l kafkaLogger) Level() kgo.LogLevel {
	// set this to the debug level so the Log method is called for every log.
	// our gosoline logger will then log according to its own log level.
	return kgo.LogLevelDebug
}

func (l kafkaLogger) Log(level kgo.LogLevel, msg string, keyvals ...any) {
	fields := map[string]any{}

	// keyvals is a slice of alternating key-value pairs providing additional information to the log.
	// the keys are always strings and the values can be of any type according to the kafka library.
	for i := 0; i < len(keyvals)-1; i += 2 {
		fields[fmt.Sprintf("%v", keyvals[i])] = keyvals[i+1]
	}

	switch level {
	case kgo.LogLevelError:
		l.logger.WithFields(fields).Error(l.ctx, msg)
	case kgo.LogLevelWarn:
		l.logger.WithFields(fields).Warn(l.ctx, msg)
	case kgo.LogLevelInfo:
		l.logger.WithFields(fields).Info(l.ctx, msg)
	case kgo.LogLevelDebug:
		l.logger.WithFields(fields).Debug(l.ctx, msg)
	default:
	}
}
