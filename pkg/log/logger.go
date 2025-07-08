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

func LevelPriority(level string) int {
	return levelPriorities[level]
}

type Data struct {
	Context       context.Context
	Channel       string
	ContextFields map[string]interface{}
	Fields        map[string]interface{}
}

type Fields map[string]interface{}

//go:generate go run github.com/vektra/mockery/v2 --name Logger
type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})

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
			ContextFields: make(map[string]interface{}),
			Fields:        make(map[string]interface{}),
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

func (l *gosoLogger) Debug(format string, args ...interface{}) {
	l.log(PriorityDebug, format, args, nil)
}

func (l *gosoLogger) Info(format string, args ...interface{}) {
	l.log(PriorityInfo, format, args, nil)
}

func (l *gosoLogger) Warn(format string, args ...interface{}) {
	l.log(PriorityWarn, format, args, nil)
}

func (l *gosoLogger) Error(format string, args ...interface{}) {
	err := fmt.Errorf(format, args...)
	msg := err.Error()

	l.log(PriorityError, "%s", []interface{}{msg}, err)
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

func (l *gosoLogger) log(level int, msg string, args []interface{}, loggedErr error) {
	timestamp := l.clock.Now()

	for _, handler := range l.handlers {
		if !l.shouldLog(l.data.Channel, level, handler) {
			continue
		}

		if handlerErr := handler.Log(timestamp, level, msg, args, loggedErr, l.data); handlerErr != nil {
			l.err(handlerErr)
		}
	}
}

func (l *gosoLogger) shouldLog(current string, level int, h Handler) bool {
	if channelLevel, ok := h.Channels()[current]; ok {
		return channelLevel <= level
	}

	return h.Level() <= level
}

func (l *gosoLogger) err(err error) {
	_, _ = fmt.Fprintf(os.Stderr, "Failed to write to log, %s\n", err)
}
