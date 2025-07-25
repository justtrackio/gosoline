package log

import (
	"context"
	"fmt"
	"math"
	"os"

	"github.com/justtrackio/gosoline/pkg/clock"
)

const (
	LevelTrace = "trace"
	LevelDebug = "debug"
	LevelInfo  = "info"
	LevelWarn  = "warn"
	LevelError = "error"
	LevelNone  = "none"

	PriorityTrace = 0
	PriorityDebug = 1
	PriorityInfo  = 2
	PriorityWarn  = 3
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

func LevelName(level int) string {
	return levelNames[level]
}

func LevelPriority(level string) (int, bool) {
	priority, ok := levelPriorities[level]

	return priority, ok
}

type Data struct {
	Context       context.Context
	Channel       string
	ContextFields map[string]any
	Fields        map[string]any
}

type Fields map[string]any

//go:generate go run github.com/vektra/mockery/v2 --name Logger
type Logger interface {
	Debug(format string, args ...any)
	Info(format string, args ...any)
	Warn(format string, args ...any)
	Error(format string, args ...any)

	WithChannel(channel string) Logger
	WithContext(ctx context.Context) Logger
	WithFields(Fields) Logger
}

type GosoLogger interface {
	Logger
	Option(opt ...Option) error
}

type LoggerSettings struct {
	Handlers map[string]HandlerSettings `cfg:"handlers"`
}

type HandlerSettings struct {
	Type string `cfg:"type"`
}

type gosoLogger struct {
	clock        clock.Clock
	data         Data
	ctxResolvers []ContextFieldsResolverFunction
	handlers     []Handler
}

func NewLogger() *gosoLogger {
	return NewLoggerWithInterfaces(clock.NewRealClock(), []Handler{})
}

func NewLoggerWithInterfaces(clock clock.Clock, handlers []Handler) *gosoLogger {
	return &gosoLogger{
		clock: clock,
		data: Data{
			Context:       nil,
			Channel:       "main",
			ContextFields: make(map[string]any),
			Fields:        make(map[string]any),
		},
		ctxResolvers: nil,
		handlers:     handlers,
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

func (l *gosoLogger) Debug(format string, args ...any) {
	l.log(PriorityDebug, format, args, nil)
}

func (l *gosoLogger) Info(format string, args ...any) {
	l.log(PriorityInfo, format, args, nil)
}

func (l *gosoLogger) Warn(format string, args ...any) {
	l.log(PriorityWarn, format, args, nil)
}

func (l *gosoLogger) Error(format string, args ...any) {
	err := fmt.Errorf(format, args...)
	msg := err.Error()

	l.log(PriorityError, "%s", []any{msg}, err)
}

func (l *gosoLogger) WithChannel(channel string) Logger {
	cpy := l.copy()
	cpy.data.Channel = channel

	return cpy
}

func (l *gosoLogger) WithContext(ctx context.Context) Logger {
	if ctx == nil {
		return l
	}

	cpy := l.copy()
	cpy.data.Context = ctx

	for _, r := range l.ctxResolvers {
		newContextFields := r(ctx)
		cpy.data.ContextFields = mergeFields(cpy.data.ContextFields, newContextFields)
	}

	return cpy
}

func (l *gosoLogger) WithFields(fields Fields) Logger {
	cpy := l.copy()
	cpy.data.Fields = mergeFields(l.data.Fields, fields)

	return cpy
}

func (l *gosoLogger) copy() *gosoLogger {
	return &gosoLogger{
		clock:        l.clock,
		data:         l.data,
		ctxResolvers: l.ctxResolvers,
		handlers:     l.handlers,
	}
}

func (l *gosoLogger) log(level int, msg string, args []any, loggedErr error) {
	timestamp := l.clock.Now()

	for _, handler := range l.handlers {
		if ok, err := l.shouldLog(l.data.Channel, level, handler); err != nil {
			l.err(err)

			continue
		} else if !ok {
			continue
		}

		if handlerErr := handler.Log(timestamp, level, msg, args, loggedErr, l.data); handlerErr != nil {
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
