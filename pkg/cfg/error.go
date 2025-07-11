package cfg

import (
	"fmt"

	"github.com/getsentry/sentry-go"
)

//go:generate go run github.com/vektra/mockery/v2 --name Sentry
type Sentry interface {
	CaptureException(exception error, hint *sentry.EventHint, scope sentry.EventModifier) *sentry.EventID
}

type ErrorHandler func(msg string, args ...any)

var defaultErrorHandler = PanicErrorHandler

func WithDefaultErrorHandler(handler ErrorHandler) {
	defaultErrorHandler = handler
}

func PanicErrorHandler(msg string, args ...any) {
	err := fmt.Errorf(msg, args...)
	panic(err)
}

func LoggerErrorHandler(logger Logger) ErrorHandler {
	return func(msg string, args ...any) {
		logger.Error(msg, args...)
	}
}

func SentryErrorHandler(sentry Sentry) ErrorHandler {
	return func(msg string, args ...any) {
		err := fmt.Errorf(msg, args...)
		sentry.CaptureException(err, nil, nil)
	}
}
