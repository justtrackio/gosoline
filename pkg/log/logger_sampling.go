package log

import (
	"context"
	"sync"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
)

type SamplingLogger struct {
	Logger
	clock    clock.Clock
	logs     *sync.Map
	interval time.Duration
}

func NewSamplingLogger(logger Logger, interval time.Duration) Logger {
	testClock := clock.NewRealClock()

	return NewSamplingLoggerWithInterfaces(logger, testClock, interval)
}

func NewSamplingLoggerWithInterfaces(logger Logger, clock clock.Clock, interval time.Duration) *SamplingLogger {
	return &SamplingLogger{
		Logger:   logger,
		clock:    clock,
		logs:     &sync.Map{},
		interval: interval,
	}
}

func (l *SamplingLogger) copy(logger Logger) *SamplingLogger {
	return &SamplingLogger{
		Logger:   logger,
		clock:    l.clock,
		logs:     l.logs,
		interval: l.interval,
	}
}

func (l *SamplingLogger) WithChannel(channel string) Logger {
	logger := l.Logger.WithChannel(channel)
	return l.copy(logger)
}

func (l *SamplingLogger) WithContext(ctx context.Context) Logger {
	logger := l.Logger.WithContext(ctx)
	return l.copy(logger)
}

func (l *SamplingLogger) WithFields(fields Fields) Logger {
	logger := l.Logger.WithFields(fields)
	return l.copy(logger)
}

func (l *SamplingLogger) Debug(msg string, args ...interface{}) {
	if !l.shouldLog(msg) {
		return
	}

	l.Logger.Debug(msg, args...)
}

func (l *SamplingLogger) Error(msg string, args ...interface{}) {
	if !l.shouldLog(msg) {
		return
	}

	l.Logger.Error(msg, args...)
}

func (l *SamplingLogger) Info(msg string, args ...interface{}) {
	if !l.shouldLog(msg) {
		return
	}

	l.Logger.Info(msg, args...)
}

func (l *SamplingLogger) Warn(msg string, args ...interface{}) {
	if !l.shouldLog(msg) {
		return
	}

	l.Logger.Warn(msg, args...)
}

func (l *SamplingLogger) shouldLog(msg string) bool {
	value, ok := l.logs.Load(msg)

	if !ok {
		lastLoggedAt := l.clock.Now()
		l.logs.Store(msg, &lastLoggedAt)

		return true
	}

	lastLoggedAt := value.(*time.Time)
	shouldLogAgainAt := lastLoggedAt.Add(l.interval)

	if shouldLogAgainAt.UnixNano() >= l.clock.Now().UnixNano() {
		return false
	}

	newLastLoggedAt := l.clock.Now()
	l.logs.Store(msg, &newLastLoggedAt)

	return true
}
