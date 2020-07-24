package mon

import (
	"context"
	"time"
)

type ContextEnforcingLogger struct {
	logger             Logger
	stacktraceProvider StackTraceProvider
	notifier           Logger
	enabled            bool
}

func NewContextEnforcingLogger(logger Logger) *ContextEnforcingLogger {
	notifier := NewSamplingLogger(logger, time.Minute)

	return NewContextEnforcingLoggerWithInterfaces(logger, GetStackTrace, notifier)
}

func NewContextEnforcingLoggerWithInterfaces(logger Logger, stacktraceProvider StackTraceProvider, notifier Logger) *ContextEnforcingLogger {
	return &ContextEnforcingLogger{
		logger:             logger,
		stacktraceProvider: stacktraceProvider,
		notifier:           notifier.WithChannel("context_missing"),
		enabled:            false,
	}
}

func (l *ContextEnforcingLogger) Enable() {
	l.enabled = true
}

func (l *ContextEnforcingLogger) Debug(args ...interface{}) {
	l.checkContext(Debug)
	l.logger.Debug(args...)
}

func (l *ContextEnforcingLogger) Debugf(msg string, args ...interface{}) {
	l.checkContext(Debug)
	l.logger.Debugf(msg, args...)
}

func (l *ContextEnforcingLogger) Error(err error, msg string) {
	l.checkContext(Error)
	l.logger.Error(err, msg)
}

func (l *ContextEnforcingLogger) Errorf(err error, msg string, args ...interface{}) {
	l.checkContext(Error)
	l.logger.Errorf(err, msg, args...)
}

func (l *ContextEnforcingLogger) Fatal(err error, msg string) {
	l.checkContext(Fatal)
	l.logger.Fatal(err, msg)
}

func (l *ContextEnforcingLogger) Fatalf(err error, msg string, args ...interface{}) {
	l.checkContext(Fatal)
	l.logger.Fatalf(err, msg, args...)
}

func (l *ContextEnforcingLogger) Info(args ...interface{}) {
	l.checkContext(Info)
	l.logger.Info(args...)
}

func (l *ContextEnforcingLogger) Infof(msg string, args ...interface{}) {
	l.checkContext(Info)
	l.logger.Infof(msg, args...)
}

func (l *ContextEnforcingLogger) Panic(err error, msg string) {
	l.checkContext(Panic)
	l.logger.Panic(err, msg)
}

func (l *ContextEnforcingLogger) Panicf(err error, msg string, args ...interface{}) {
	l.checkContext(Panic)
	l.logger.Panicf(err, msg, args...)
}

func (l *ContextEnforcingLogger) Warn(args ...interface{}) {
	l.checkContext(Warn)
	l.logger.Warn(args...)
}

func (l *ContextEnforcingLogger) Warnf(msg string, args ...interface{}) {
	l.checkContext(Warn)
	l.logger.Warnf(msg, args...)
}

func (l *ContextEnforcingLogger) WithChannel(channel string) Logger {
	return &ContextEnforcingLogger{
		logger:             l.logger.WithChannel(channel),
		stacktraceProvider: l.stacktraceProvider,
		notifier:           l.notifier,
		enabled:            l.enabled,
	}
}

func (l *ContextEnforcingLogger) WithContext(ctx context.Context) Logger {
	// we can stop wrapping that logger, it now carries a context
	return l.logger.WithContext(ctx)
}

func (l *ContextEnforcingLogger) WithFields(fields map[string]interface{}) Logger {
	return &ContextEnforcingLogger{
		logger:             l.logger.WithFields(fields),
		stacktraceProvider: l.stacktraceProvider,
		notifier:           l.notifier,
		enabled:            l.enabled,
	}
}

func (l *ContextEnforcingLogger) checkContext(level string) {
	if !l.enabled {
		return
	}

	base, ok := l.logger.(*logger)

	if !ok {
		l.notifier.Warnf("context enforcing logger wrapping something else than *mon.logger: %T", l.logger)

		return
	}

	levelNo := levels[level]

	if levelNo < base.level {
		return
	}

	if base.data.Context != nil {
		return
	}

	stacktrace := l.stacktraceProvider(1)

	l.notifier.Warn("you should add the context to your logger:", stacktrace)
}
