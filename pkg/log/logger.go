package log

import "context"

type Fields map[string]interface{}

//go:generate mockery -name Logger
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})

	WithChannel(channel string) Logger
	WithContext(ctx context.Context) Logger
	WithFields(fields Fields) Logger
}
