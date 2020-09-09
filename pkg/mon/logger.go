package mon

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"os"
	"reflect"
	"sync"
	"time"
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

func AllLogLevels() []string {
	return []string{
		Trace,
		Debug,
		Info,
		Warn,
		Error,
		Fatal,
		Panic,
	}
}

var levels = map[string]int{
	Trace: 0,
	Debug: 1,
	Info:  2,
	Warn:  3,
	Error: 4,
	Fatal: 5,
	Panic: 6,
}

func levelPriority(level string) int {
	return levels[level]
}

const (
	ChannelDefault   = "default"
	FormatConsole    = "console"
	FormatGelf       = "gelf"
	FormatGelfFields = "gelf_fields"
	FormatJson       = "json"
)

type HandlerFactory func(config cfg.Config, name string) (Handler, error)

type Handler interface {
	Write(level string, msg string, logErr error, metadata Metadata)
}

var HandlerFactories = map[string]HandlerFactory{}

type Tags map[string]interface{}
type ConfigValues map[string]interface{}
type Fields map[string]interface{}
type EcsMetadata map[string]interface{}

type Metadata struct {
	Channel       string
	Context       context.Context
	ContextFields Fields
	Fields        Fields
	Tags          Tags
}

type formatter func(timestamp string, level string, msg string, err error, data *Metadata) ([]byte, error)

var formatters = map[string]formatter{
	FormatConsole:    formatterConsole,
	FormatGelf:       formatterGelf,
	FormatGelfFields: formatterGelfFields,
	FormatJson:       formatterJson,
}

type GosoLog interface {
	Logger
	Option(options ...LoggerOption) error
}

//go:generate mockery -name Logger
type Logger interface {
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(err error, msg string)
	Panic(err error, msg string)
	Fatal(err error, msg string)

	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(err error, format string, args ...interface{})
	Panicf(err error, format string, args ...interface{})
	Fatalf(err error, format string, args ...interface{})

	WithChannel(channel string) Logger
	WithContext(ctx context.Context) Logger
	WithFields(fields Fields) Logger
}

type logger struct {
	outputLck   *sync.Mutex
	ctxResolver []ContextFieldsResolver
	handlers    []Handler
	hooks       []LoggerHook

	level int

	data Metadata
}

func NewLogger() *logger {
	return NewLoggerWithInterfaces()
}

func NewLoggerWithInterfaces() *logger {
	logger := &logger{
		outputLck:   &sync.Mutex{},
		ctxResolver: make([]ContextFieldsResolver, 0),
		handlers:    make([]Handler, 0),
		hooks:       make([]LoggerHook, 0),
		level:       levelPriority(Info),
		data: Metadata{
			Channel:       ChannelDefault,
			ContextFields: make(Fields),
			Fields:        make(Fields),
			Tags:          make(Tags),
		},
	}

	return logger
}

func (l *logger) copy() *logger {
	return &logger{
		outputLck:   l.outputLck,
		ctxResolver: l.ctxResolver,
		handlers:    l.handlers,
		hooks:       l.hooks,
		level:       l.level,
		data:        l.data,
	}
}

func (l *logger) Option(options ...LoggerOption) error {
	for _, opt := range options {
		if err := opt(l); err != nil {
			return err
		}
	}

	return nil
}

func (l *logger) WithChannel(channel string) Logger {
	cpy := l.copy()
	cpy.data.Channel = channel

	return cpy
}

func (l *logger) WithContext(ctx context.Context) Logger {
	if ctx == nil {
		return l
	}

	cpy := l.copy()
	cpy.data.Context = ctx

	for _, r := range l.ctxResolver {
		newContextFields := r(ctx)
		cpy.data.ContextFields = mergeMapStringInterface(cpy.data.ContextFields, newContextFields)
	}

	return cpy
}

func (l *logger) WithFields(fields Fields) Logger {
	cpy := l.copy()
	cpy.data.Fields = mergeMapStringInterface(l.data.Fields, fields)

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
	l.log(level, msg, err, Fields{
		"stacktrace": GetStackTrace(1),
	})
}

func (l *logger) log(level string, msg string, logErr error, fields Fields) {
	levelNo := levels[level]

	if levelNo < l.level {
		return
	}

	cpyData := l.data
	cpyData.Fields = mergeMapStringInterface(cpyData.Fields, fields)

	for _, h := range l.hooks {
		if err := h.Fire(level, msg, logErr, &cpyData); err != nil {
			l.err(err)
		}
	}

	for _, handler := range l.handlers {
		handler.Write(level, msg, logErr, cpyData)
	}
}

func (l *logger) err(err error) {
	l.outputLck.Lock()
	defer l.outputLck.Unlock()

	for _, handler := range l.handlers {
		handler.Write(Error, err.Error(), err, l.data)
	}
}

func mergeMapStringInterface(receiver map[string]interface{}, input map[string]interface{}) map[string]interface{} {
	newMap := make(map[string]interface{}, len(receiver)+len(input))

	for k, v := range receiver {
		newMap[k] = prepareForLog(v)
	}

	for k, v := range input {
		newMap[k] = prepareForLog(v)
	}

	return newMap
}

func prepareForLog(v interface{}) interface{} {
	switch t := v.(type) {
	case error:
		// Otherwise errors are ignored by `encoding/json`
		return t.Error()
	case time.Time:
		return v
	case map[string]interface{}:
		// perform a deep copy of any maps contained in this map element
		// to ensure we own the object completely
		return mergeMapStringInterface(t, nil)

	default:
		// same as before, but handle the case of the map mapping to something
		// different than interface{}
		// should quite rarely get hit, otherwise you are using too complex objects for your logs
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Map:
			iter := rv.MapRange()
			newMap := make(map[string]interface{}, rv.Len())

			for iter.Next() {
				keyValue := iter.Key()
				elemValue := iter.Value()
				newMap[fmt.Sprint(keyValue.Interface())] = prepareForLog(elemValue.Interface())
			}

			return newMap

		case reflect.Ptr, reflect.Interface:
			if rv.IsNil() {
				return nil
			}

			return prepareForLog(rv.Elem().Interface())

		case reflect.Struct:
			rvt := rv.Type()
			newMap := make(map[string]interface{}, rv.NumField())

			for i := 0; i < rv.NumField(); i++ {
				field := rv.Field(i)
				if !field.CanInterface() {
					continue
				}
				newMap[rvt.Field(i).Name] = prepareForLog(field.Interface())
			}

			return newMap

		case reflect.Slice, reflect.Array:
			if rv.Kind() == reflect.Slice && rv.IsNil() {
				return nil
			}

			newArray := make([]interface{}, rv.Len())

			for i := range newArray {
				newArray[i] = prepareForLog(rv.Index(i).Interface())
			}

			return newArray

		default:
			return v
		}
	}
}
