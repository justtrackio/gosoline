package cfg

import (
	"fmt"

	"github.com/getsentry/sentry-go"
)

//go:generate mockery --name Sentry
type Sentry interface {
	CaptureException(exception error, hint *sentry.EventHint, scope sentry.EventModifier) *sentry.EventID
}

type ErrorHandler func(msg string, args ...interface{})

var defaultErrorHandler = PanicErrorHandler

func WithDefaultErrorHandler(handler ErrorHandler) {
	defaultErrorHandler = handler
}

func PanicErrorHandler(msg string, args ...interface{}) {
	err := fmt.Errorf(msg, args...)
	panic(err)
}

func LoggerErrorHandler(logger Logger) ErrorHandler {
	return func(msg string, args ...interface{}) {
		logger.Error(msg, args...)
	}
}

func SentryErrorHandler(sentry Sentry) ErrorHandler {
	return func(msg string, args ...interface{}) {
		err := fmt.Errorf(msg, args...)
		sentry.CaptureException(err, nil, nil)
	}
}
