package aws

import (
	"context"

	"github.com/aws/smithy-go/logging"
	"github.com/justtrackio/gosoline/pkg/log"
)

type Logger struct {
	base log.Logger
}

func NewLogger(base log.Logger) *Logger {
	return &Logger{
		base: base,
	}
}

func (l Logger) Logf(classification logging.Classification, format string, v ...interface{}) {
	switch classification {
	case logging.Warn:
		l.base.Warn(format, v...)
	default:
		l.base.Info(format, v...)
	}
}

func (l Logger) WithContext(ctx context.Context) logging.Logger {
	return &Logger{
		base: l.base.WithContext(ctx),
	}
}
