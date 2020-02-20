package mon

import (
	"time"
)

type ContextEnforcingLogger struct {
	Logger
	stacktraceProvider StackTraceProvider
	notifier           Logger
	enabled            bool
}

func NewContextEnforcingLogger(logger Logger) *ContextEnforcingLogger {
	notifier := NewSamplingLogger(logger, time.Minute)

	return &ContextEnforcingLogger{
		Logger:             logger,
		stacktraceProvider: GetStackTrace,
		notifier:           notifier,
		enabled:            false,
	}
}

func (l *ContextEnforcingLogger) Enable() {
	l.enabled = true
}

func (l *ContextEnforcingLogger) Debug(args ...interface{}) {
	l.checkContext()
	l.Logger.Debug(args...)
}

func (l *ContextEnforcingLogger) Debugf(msg string, args ...interface{}) {
	l.checkContext()
	l.Logger.Debugf(msg, args...)
}

func (l *ContextEnforcingLogger) Error(err error, msg string) {
	l.checkContext()
	l.Logger.Error(err, msg)
}

func (l *ContextEnforcingLogger) Errorf(err error, msg string, args ...interface{}) {
	l.checkContext()
	l.Logger.Errorf(err, msg, args...)
}

func (l *ContextEnforcingLogger) Fatal(err error, msg string) {
	l.checkContext()
	l.Logger.Fatal(err, msg)
}

func (l *ContextEnforcingLogger) Fatalf(err error, msg string, args ...interface{}) {
	l.checkContext()
	l.Logger.Fatalf(err, msg, args...)
}

func (l *ContextEnforcingLogger) Info(args ...interface{}) {
	l.checkContext()
	l.Logger.Info(args...)
}

func (l *ContextEnforcingLogger) Infof(msg string, args ...interface{}) {
	l.checkContext()
	l.Logger.Infof(msg, args...)
}

func (l *ContextEnforcingLogger) Panic(err error, msg string) {
	l.checkContext()
	l.Logger.Panic(err, msg)
}

func (l *ContextEnforcingLogger) Panicf(err error, msg string, args ...interface{}) {
	l.checkContext()
	l.Logger.Panicf(err, msg, args...)
}

func (l *ContextEnforcingLogger) Warn(args ...interface{}) {
	l.checkContext()
	l.Logger.Warn(args...)
}

func (l *ContextEnforcingLogger) Warnf(msg string, args ...interface{}) {
	l.checkContext()
	l.Logger.Warnf(msg, args...)
}

func (l *ContextEnforcingLogger) checkContext() {
	if !l.enabled {
		return
	}

	base, ok := l.Logger.(*logger)

	if !ok {
		return
	}

	if base.data.Context != nil {
		return
	}

	stacktrace := l.stacktraceProvider(1)

	l.notifier.Warn("you should add the context to your logger:", stacktrace)
}
