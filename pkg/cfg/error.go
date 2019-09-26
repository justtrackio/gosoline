package cfg

import (
	"github.com/getsentry/raven-go"
	"github.com/pkg/errors"
)

//go:generate mockery -name Sentry
type Sentry interface {
	CaptureErrorAndWait(err error, tags map[string]string, interfaces ...raven.Interface) string
}

type Logger interface {
	Errorf(err error, msg string, args ...interface{})
}

type ErrorHandler func(err error, msg string, args ...interface{})

var defaultErrorHandler = PanicErrorHandler

func WithDefaultErrorHandler(handler ErrorHandler) {
	defaultErrorHandler = handler
}

func PanicErrorHandler(err error, msg string, args ...interface{}) {
	err = errors.Wrapf(err, msg, args...)
	panic(err)
}

func LoggerErrorHandler(logger Logger) ErrorHandler {
	return func(err error, msg string, args ...interface{}) {
		logger.Errorf(err, msg, args...)
	}
}

func SentryErrorHandler(sentry Sentry) ErrorHandler {
	return func(err error, msg string, args ...interface{}) {
		err = errors.Wrapf(err, msg, args...)
		sentry.CaptureErrorAndWait(err, nil)
	}
}
