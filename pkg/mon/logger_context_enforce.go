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

func (l *ContextEnforcingLogger) Debug(msg string, args ...interface{}) {
	l.checkContext(Debug)
	l.logger.Debug(msg, args...)
}

func (l *ContextEnforcingLogger) Error(msg string, args ...interface{}) {
	l.checkContext(Error)
	l.logger.Error(msg, args...)
}

func (l *ContextEnforcingLogger) Info(msg string, args ...interface{}) {
	l.checkContext(Info)
	l.logger.Info(msg, args...)
}

func (l *ContextEnforcingLogger) Warn(msg string, args ...interface{}) {
	l.checkContext(Warn)
	l.logger.Warn(msg, args...)
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

func (l *ContextEnforcingLogger) WithFields(fields Fields) Logger {
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
		l.notifier.Warn("context enforcing logger wrapping something else than *mon.logger: %T", l.logger)

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

	l.notifier.Warn("you should add the context to your logger: %s", stacktrace)
}
