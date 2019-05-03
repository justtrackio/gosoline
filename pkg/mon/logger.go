package mon

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/getsentry/raven-go"
	"github.com/jonboulle/clockwork"
	"io"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

const (
	Trace = "trace"
	Debug = "debug"
	Info  = "info"
	Warn  = "warn"
	Error = "error"
	Fatal = "fatal"
	Panic = "panic"
)

var levels = map[string]int{
	Trace: 0,
	Debug: 1,
	Info:  2,
	Warn:  3,
	Error: 4,
	Fatal: 5,
	Panic: 6,
}

func levelPrio(level string) int {
	return levels[level]
}

const (
	ChannelDefault   = "default"
	FormatConsole    = "console"
	FormatGelf       = "gelf"
	FormatGelfFields = "gelf_fields"
	FormatJson       = "json"
)

type Tags map[string]string
type TagsFromConfig map[string]string
type ConfigValues map[string]interface{}
type Fields map[string]interface{}
type EcsMetadata map[string]interface{}

type formatter func(clock clockwork.Clock, channel string, level string, msg string, logErr error, fields Fields) ([]byte, error)

var formatters = map[string]formatter{
	FormatConsole:    formatterConsole,
	FormatGelf:       formatterGelf,
	FormatGelfFields: formatterGelfFields,
	FormatJson:       formatterJson,
}

//go:generate mockery -name Sentry
type Sentry interface {
	Capture(packet *raven.Packet, captureTags map[string]string) (eventID string, ch chan error)
}

//go:generate mockery -name Logger
type Logger interface {
	WithChannel(channel string) Logger
	WithContext(ctx context.Context) Logger
	WithFields(fields map[string]interface{}) Logger
	Info(args ...interface{})
	Infof(msg string, args ...interface{})
	Debug(args ...interface{})
	Debugf(msg string, args ...interface{})
	Warn(args ...interface{})
	Warnf(msg string, args ...interface{})
	Error(err error, msg string)
	Errorf(err error, msg string, args ...interface{})
	Fatal(err error, msg string)
	Fatalf(err error, msg string, args ...interface{})
	Panic(err error, msg string)
	Panicf(err error, msg string, args ...interface{})
}

type logger struct {
	clock     clockwork.Clock
	output    io.Writer
	outputLck *sync.Mutex
	hooks     []LoggerHook

	format string
	level  int

	tags         Tags
	configValues ConfigValues
	channel      string
	fields       Fields
	context      context.Context

	ecsLck       *sync.Mutex
	ecsAvailable bool
	ecsMetadata  EcsMetadata
}

func (l *logger) copy() *logger {
	return &logger{
		clock:        l.clock,
		outputLck:    l.outputLck,
		output:       l.output,
		hooks:        l.hooks,
		level:        l.level,
		format:       l.format,
		tags:         l.tags,
		configValues: l.configValues,
		channel:      l.channel,
		fields:       l.fields,
		context:      l.context,
		ecsLck:       l.ecsLck,
		ecsAvailable: l.ecsAvailable,
		ecsMetadata:  l.ecsMetadata,
	}
}

func (l *logger) addHook(hook LoggerHook) {
	l.hooks = append(l.hooks, hook)
}

func (l *logger) WithChannel(channel string) Logger {
	cpy := l.copy()
	cpy.channel = channel

	return cpy
}

func (l *logger) WithContext(ctx context.Context) Logger {
	span := tracing.GetSpan(ctx)

	if span == nil {
		return l
	}

	cpy := l.copy()
	cpy.context = ctx

	return cpy.WithFields(Fields{
		"trace_id": span.GetTrace().GetTraceId(),
	})
}

func (l *logger) WithFields(fields map[string]interface{}) Logger {
	newFields := make(Fields, len(l.fields)+len(fields))

	for k, v := range l.fields {
		newFields[k] = v
	}

	for k, v := range fields {
		newFields[k] = v
	}

	cpy := l.copy()
	cpy.fields = newFields

	return cpy
}

func (l *logger) Info(args ...interface{}) {
	l.log(Info, fmt.Sprint(args...), nil, Fields{})
}

func (l *logger) Infof(msg string, args ...interface{}) {
	l.log(Info, fmt.Sprintf(msg, args...), nil, Fields{})
}

func (l *logger) Debug(args ...interface{}) {
	if l.level > levels[Debug] {
		return
	}

	l.log(Debug, fmt.Sprint(args...), nil, Fields{})
}

func (l *logger) Debugf(msg string, args ...interface{}) {
	if l.level > levels[Debug] {
		return
	}

	l.log(Debug, fmt.Sprintf(msg, args...), nil, Fields{})
}

func (l *logger) Warn(args ...interface{}) {
	l.log(Warn, fmt.Sprint(args...), nil, Fields{})
}

func (l *logger) Warnf(msg string, args ...interface{}) {
	l.log(Warn, fmt.Sprintf(msg, args...), nil, Fields{})
}

func (l *logger) Error(err error, msg string) {
	l.logError(Error, err, msg)
}

func (l *logger) Errorf(err error, msg string, args ...interface{}) {
	l.logError(Error, err, fmt.Sprintf(msg, args...))
}

func (l *logger) Fatal(err error, msg string) {
	l.logError(Fatal, err, msg)
	os.Exit(1)
}

func (l *logger) Fatalf(err error, msg string, args ...interface{}) {
	l.logError(Fatal, err, fmt.Sprintf(msg, args...))
	os.Exit(1)
}

func (l *logger) Panic(err error, msg string) {
	l.logError(Panic, err, msg)
	panic(err)
}

func (l *logger) Panicf(err error, msg string, args ...interface{}) {
	l.logError(Panic, err, fmt.Sprintf(msg, args...))
	panic(err)
}

func (l *logger) logError(level string, err error, msg string) {
	if l.context != nil {
		span := tracing.GetSpan(l.context)

		if span != nil {
			span.AddError(err)
		}
	}

	l.log(level, msg, err, Fields{
		"stacktrace": getStackTrace(1),
	})
}

func (l *logger) log(level string, msg string, logErr error, fields Fields) {
	levelNo := levels[level]

	if levelNo < l.level {
		return
	}

	ecsMetadata := l.readEcsMetadata()

	for _, h := range l.hooks {
		h.Fire(level, msg, logErr, l.fields, l.tags, l.configValues, l.context, ecsMetadata)
	}

	for k, v := range l.fields {
		switch v := v.(type) {
		case error:
			// Otherwise errors are ignored by `encoding/json`
			fields[k] = v.Error()
		default:
			fields[k] = v
		}
	}

	buffer, err := formatters[l.format](l.clock, l.channel, level, msg, logErr, fields)

	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to write to log, %v\n", err)
	}

	l.write(buffer)
}

func (l *logger) write(buffer []byte) {
	l.outputLck.Lock()
	defer l.outputLck.Unlock()

	_, err := l.output.Write(buffer)

	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to write to log, %v\n", err)
	}
}

func round(val float64, places int) float64 {
	shift := math.Pow(10, float64(places))
	return math.Floor((val*shift)+.5) / shift
}

// getStackTrace constructs the current stacktrace. depthSkip defines how many steps of the
// stacktrace should be skipped. This is useful to not clutter the stacktrace with logging
// function calls.
func getStackTrace(depthSkip int) string {
	depthSkip = depthSkip + 1 // Skip this function in stacktrace
	maxDepth := 50
	traces := make([]string, maxDepth)

	// Get traces
	var depth int
	for depth = 0; depth < maxDepth; depth++ {
		function, _, line, ok := runtime.Caller(depth)

		if !ok {
			break
		}

		var traceStrBuilder strings.Builder
		traceStrBuilder.WriteString("\t")
		traceStrBuilder.WriteString(runtime.FuncForPC(function).Name())
		traceStrBuilder.WriteString(":")
		traceStrBuilder.WriteString(strconv.Itoa(line))
		traceStrBuilder.WriteString("\n")
		traces[depth] = traceStrBuilder.String()
	}

	// Assemble stacktrace in reverse order
	var strBuilder strings.Builder
	strBuilder.WriteString("\n")
	for i := depth; i > depthSkip; i-- {
		strBuilder.WriteString(traces[i])
	}
	return strBuilder.String()
}
