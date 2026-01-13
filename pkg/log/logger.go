package log

import (
	"context"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/smpl/smplctx"
)

const (
	// LevelTrace is the lowest priority (most verbose) log level.
	LevelTrace = "trace"
	// LevelDebug indicates information useful for developers.
	LevelDebug = "debug"
	// LevelInfo is the default level for operational logs.
	LevelInfo = "info"
	// LevelWarn indicates recoverable issues.
	LevelWarn = "warn"
	// LevelError indicates failures requiring attention.
	LevelError = "error"
	// LevelNone disables logging.
	LevelNone = "none"

	// PriorityTrace is the numeric priority for trace logs (0).
	PriorityTrace = 0
	// PriorityDebug is the numeric priority for debug logs (1).
	PriorityDebug = 1
	// PriorityInfo is the numeric priority for info logs (2).
	PriorityInfo = 2
	// PriorityWarn is the numeric priority for warn logs (3).
	PriorityWarn = 3
	// PriorityError is the numeric priority for error logs (4).
	PriorityError = 4
	// PriorityNone is used to indicate that no logging should be done.
	// Value is set to the maximum int value to ensure that it is always greater than any other priority.
	PriorityNone = math.MaxInt
)

var levelNames = map[int]string{
	PriorityTrace: LevelTrace,
	PriorityDebug: LevelDebug,
	PriorityInfo:  LevelInfo,
	PriorityWarn:  LevelWarn,
	PriorityError: LevelError,
	PriorityNone:  LevelNone,
}

var levelPriorities = map[string]int{
	LevelTrace: PriorityTrace,
	LevelDebug: PriorityDebug,
	LevelInfo:  PriorityInfo,
	LevelWarn:  PriorityWarn,
	LevelError: PriorityError,
	LevelNone:  PriorityNone,
}

// LevelName returns the string representation of a log level priority (e.g., 2 -> "info").
func LevelName(level int) string {
	return levelNames[level]
}

// LevelPriority returns the numeric priority for a given log level name (e.g., "info" -> 2).
// It returns false if the level name is unknown.
func LevelPriority(level string) (int, bool) {
	priority, ok := levelPriorities[level]

	return priority, ok
}

// Data holds the structured context for a log entry, including channel, fields, and context-derived fields.
type Data struct {
	Channel       string
	ContextFields map[string]any
	Fields        map[string]any
}

// Fields is a map of key-value pairs to add structured data to a log entry.
type Fields map[string]any

// Logger is the main interface for logging. It supports standard log levels (Debug, Info, Warn, Error)
// and methods to create derived loggers with specific channels or fields.
//
//go:generate go run github.com/vektra/mockery/v2 --name Logger
type Logger interface {
	Debug(ctx context.Context, format string, args ...any)
	Info(ctx context.Context, format string, args ...any)
	Warn(ctx context.Context, format string, args ...any)
	Error(ctx context.Context, format string, args ...any)

	WithChannel(channel string) Logger
	WithFields(Fields) Logger
}

// GosoLogger extends the Logger interface with the ability to apply functional options after creation.
type GosoLogger interface {
	Logger
	Option(opt ...Option) error
}

// LoggerSettings holds the configuration for the main logger, specifically its handlers.
type LoggerSettings struct {
	Handlers map[string]HandlerSettings `cfg:"handlers"`
}

// HandlerSettings defines the configuration for a single log handler (e.g., its type like "iowriter" or "sentry").
type HandlerSettings struct {
	Type string `cfg:"type"`
}

var _ Logger = &gosoLogger{}

type gosoLogger struct {
	clock           clock.Clock
	data            Data
	ctxResolvers    []ContextFieldsResolverFunction
	handlers        []Handler
	samplingEnabled bool
}

// NewLogger creates a new logger with a real clock and no handlers.
func NewLogger() *gosoLogger {
	return NewLoggerWithInterfaces(clock.NewRealClock(), []Handler{})
}

// NewLoggerWithInterfaces creates a new logger with the provided clock and handlers.
func NewLoggerWithInterfaces(clock clock.Clock, handlers []Handler) *gosoLogger {
	return &gosoLogger{
		clock: clock,
		data: Data{
			Channel:       "main",
			ContextFields: make(map[string]any),
			Fields:        make(map[string]any),
		},
		ctxResolvers:    nil,
		handlers:        handlers,
		samplingEnabled: false,
	}
}

func (l *gosoLogger) Option(options ...Option) error {
	for _, opt := range options {
		if err := opt(l); err != nil {
			return fmt.Errorf("can not apply option %T: %w", opt, err)
		}
	}

	return nil
}

func (l *gosoLogger) Debug(ctx context.Context, format string, args ...any) {
	l.log(ctx, PriorityDebug, format, args, nil)
}

func (l *gosoLogger) Info(ctx context.Context, format string, args ...any) {
	l.log(ctx, PriorityInfo, format, args, nil)
}

func (l *gosoLogger) Warn(ctx context.Context, format string, args ...any) {
	l.log(ctx, PriorityWarn, format, args, nil)
}

func (l *gosoLogger) Error(ctx context.Context, format string, args ...any) {
	err := fmt.Errorf(format, args...)
	msg := err.Error()

	l.log(ctx, PriorityError, "%s", []any{msg}, err)
}

func (l *gosoLogger) WithChannel(channel string) Logger {
	cpy := l.copy()
	cpy.data.Channel = channel

	return cpy
}

func (l *gosoLogger) WithFields(fields Fields) Logger {
	cpy := l.copy()
	cpy.data.Fields = mergeFields(l.data.Fields, fields)

	return cpy
}

func (l *gosoLogger) copy() *gosoLogger {
	return &gosoLogger{
		clock:           l.clock,
		data:            l.data,
		ctxResolvers:    l.ctxResolvers,
		handlers:        l.handlers,
		samplingEnabled: l.samplingEnabled,
	}
}

func (l *gosoLogger) log(ctx context.Context, level int, msg string, args []any, loggedErr error) {
	timestamp := l.clock.Now()

	data := &Data{
		Channel:       l.data.Channel,
		ContextFields: make(map[string]any),
		Fields:        l.data.Fields,
	}

	for _, r := range l.ctxResolvers {
		newContextFields := r(ctx)
		data.ContextFields = mergeFields(data.ContextFields, newContextFields)
	}

	if l.samplingEnabled && !smplctx.IsSampled(ctx) {
		l.executeFingersCrossed(ctx, timestamp, level, msg, args, loggedErr, data)

		return
	}

	l.executeHandlers(ctx, timestamp, level, msg, args, loggedErr, data)
}

func (l *gosoLogger) executeFingersCrossed(ctx context.Context, timestamp time.Time, level int, msg string, args []any, loggedErr error, data *Data) {
	scope := getFingersCrossedScope(ctx)

	if scope == nil && level >= PriorityError {
		l.executeHandlers(ctx, timestamp, level, msg, args, loggedErr, data)

		return
	}

	if scope == nil {
		return
	}

	appendToFingersCrossedScope(ctx, l, timestamp, level, msg, args, loggedErr, data)

	if scope.flushed || level >= PriorityError {
		scope.flush()
	}
}

func (l *gosoLogger) executeHandlers(ctx context.Context, timestamp time.Time, level int, msg string, args []any, loggedErr error, data *Data) {
	for _, handler := range l.handlers {
		if ok, err := l.shouldLog(l.data.Channel, level, handler); err != nil {
			l.err(err)

			continue
		} else if !ok {
			continue
		}

		if handlerErr := handler.Log(ctx, timestamp, level, msg, args, loggedErr, *data); handlerErr != nil {
			l.err(handlerErr)
		}
	}
}

func (l *gosoLogger) shouldLog(current string, level int, h Handler) (bool, error) {
	if channelLevel, err := h.ChannelLevel(current); err != nil {
		return false, fmt.Errorf("can not get channel level: %w", err)
	} else if channelLevel != nil {
		return *channelLevel <= level, nil
	}

	return h.Level() <= level, nil
}

func (l *gosoLogger) err(err error) {
	_, _ = fmt.Fprintf(os.Stderr, "Failed to write to log, %s\n", err)
}
