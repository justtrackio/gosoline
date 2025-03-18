package mocks

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/stretchr/objx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type Mock interface {
	String() string
	TestData() objx.Map
	Test(t mock.TestingT)
	On(methodName string, arguments ...interface{}) *mock.Call
	Called(arguments ...interface{}) mock.Arguments
	MethodCalled(methodName string, arguments ...interface{}) mock.Arguments
	AssertExpectations(t mock.TestingT) bool
	AssertNumberOfCalls(t mock.TestingT, methodName string, expectedCalls int) bool
	AssertCalled(t mock.TestingT, methodName string, arguments ...interface{}) bool
	AssertNotCalled(t mock.TestingT, methodName string, arguments ...interface{}) bool
	IsMethodCallable(t mock.TestingT, methodName string, arguments ...interface{}) bool
}

type LoggerMock interface {
	log.Logger
	Mock
	EXPECT() *Logger_Expecter
}

type loggerMockOptions struct {
	t              *testing.T
	mockUntilLevel *int
}

type LoggerMockOption func(*loggerMockOptions)

type loggerMock struct {
	*Logger
	t              *testing.T
	currentChannel string
	currentFields  log.Fields
	lck            *sync.Mutex
	pendingLogs    map[string][]pendingLogMessage
}

type pendingLogMessage struct {
	message   string
	level     string
	channel   string
	fields    log.Fields
	timestamp time.Time
}

func (l *loggerMock) WithChannel(channel string) log.Logger {
	// forward potential calls to the underlying mock if we expect some
	if _, ok := funk.FindFirstFunc(l.Logger.ExpectedCalls, func(call *mock.Call) bool {
		return call.Method == "WithChannel"
	}); ok {
		l.Logger.WithChannel(channel)
	}

	return &loggerMock{
		Logger:         l.Logger,
		t:              l.t,
		currentChannel: channel,
		currentFields:  l.currentFields,
		lck:            l.lck,
		pendingLogs:    l.pendingLogs,
	}
}

func (l *loggerMock) WithContext(ctx context.Context) log.Logger {
	// forward potential calls to the underlying mock if we expect some
	if _, ok := funk.FindFirstFunc(l.Logger.ExpectedCalls, func(call *mock.Call) bool {
		return call.Method == "WithContext"
	}); ok {
		l.Logger.WithContext(ctx)
	}

	contextFields := log.ContextFieldsResolver(ctx)

	return l.WithFields(contextFields)
}

func (l *loggerMock) WithFields(fields log.Fields) log.Logger {
	// forward potential calls to the underlying mock if we expect some
	if _, ok := funk.FindFirstFunc(l.Logger.ExpectedCalls, func(call *mock.Call) bool {
		return call.Method == "WithFields"
	}); ok {
		l.Logger.WithFields(fields)
	}

	return &loggerMock{
		Logger:         l.Logger,
		t:              l.t,
		currentChannel: l.currentChannel,
		currentFields:  funk.MergeMaps(l.currentFields, fields),
		lck:            l.lck,
		pendingLogs:    l.pendingLogs,
	}
}

// WithTestingT creates a LoggerMockOption that supplies the testing.T value to use for the logger. This enables the logger to fail the test instead
// of panicking (which could be caught) if a non-mocked log level is used, print the logs in case of a failed test after the test, and automatically
// assert any expectations for the created mock.
func WithTestingT(t *testing.T) LoggerMockOption {
	return func(options *loggerMockOptions) {
		options.t = t
	}
}

// WithMockUntilLevel creates a LoggerMockOption that mocks calls up to the given log level. All other calls will cause an error and fail the test.
func WithMockUntilLevel(level int) LoggerMockOption {
	return func(options *loggerMockOptions) {
		options.mockUntilLevel = &level
	}
}

// WithMockAll is a LoggerMockOption that mocks calls to all log levels.
func WithMockAll(options *loggerMockOptions) {
	options.mockUntilLevel = mdl.Box(log.PriorityError)
}

// NewLoggerMock creates a new logger mock with the given options.
func NewLoggerMock(opts ...LoggerMockOption) LoggerMock {
	var options loggerMockOptions
	for _, opt := range opts {
		opt(&options)
	}

	var baseLogger *Logger
	if options.t != nil {
		baseLogger = NewLogger(options.t)
	} else {
		baseLogger = new(Logger)
	}

	logger := &loggerMock{
		Logger:         baseLogger,
		t:              options.t,
		currentChannel: "main",
		currentFields:  log.Fields{},
		lck:            &sync.Mutex{},
		pendingLogs:    map[string][]pendingLogMessage{},
	}

	if logger.t != nil {
		logger.t.Cleanup(func() {
			if !logger.t.Failed() {
				return
			}

			logger.printLogs()
		})
	}

	if options.mockUntilLevel != nil {
		logger.mockLoggerMethod("Debug", log.LevelDebug, *options.mockUntilLevel >= log.PriorityDebug)
		logger.mockLoggerMethod("Info", log.LevelInfo, *options.mockUntilLevel >= log.PriorityInfo)
		logger.mockLoggerMethod("Warn", log.LevelWarn, *options.mockUntilLevel >= log.PriorityWarn)
		logger.mockLoggerMethod("Error", log.LevelError, *options.mockUntilLevel >= log.PriorityError)
	}

	return logger
}

// NewLoggerMockedAll is the same as NewLoggerMock(WithMockAll).
//
// Deprecated: use NewLoggerMock(WithMockAll) instead.
func NewLoggerMockedAll(opts ...LoggerMockOption) LoggerMock {
	return NewLoggerMock(append([]LoggerMockOption{WithMockAll}, opts...)...)
}

// NewLoggerMockedUntilLevel returns a logger mocked up to the given log level. All other calls will cause an error and fail the test.
//
// Deprecated: use NewLoggerMock(WithMockUntilLevel(level)) instead.
func NewLoggerMockedUntilLevel(level int, opts ...LoggerMockOption) LoggerMock {
	return NewLoggerMock(append([]LoggerMockOption{WithMockUntilLevel(level)}, opts...)...)
}

func (l *loggerMock) mockLoggerMethod(method string, level string, allowed bool) {
	anythings := make(mock.Arguments, 0)
	f := l.inspectLogFunction(level, allowed)

	for i := 0; i < 10; i++ {
		anythings = append(anythings, mock.Anything)
		l.On(method, anythings...).Run(f).Return(l).Maybe()
	}
}

func (l *loggerMock) inspectLogFunction(level string, allowed bool) func(args mock.Arguments) {
	return func(args mock.Arguments) {
		msg := args.Get(0).(string)
		msg = fmt.Sprintf(msg, args[1:]...)

		if l.t != nil {
			testName := l.t.Name()

			l.lck.Lock()
			l.pendingLogs[testName] = append(l.pendingLogs[testName], pendingLogMessage{
				message:   msg,
				level:     level,
				channel:   l.currentChannel,
				fields:    l.currentFields,
				timestamp: time.Now().UTC(),
			})
			l.lck.Unlock()
		}

		if !allowed {
			if l.t != nil {
				l.t.Fatalf("invalid log message %q. Logs of level %s are not allowed", msg, level)
			} else {
				panic(fmt.Errorf("invalid log message %q. Logs of level %s are not allowed", msg, level))
			}
		}
	}
}

func (l *loggerMock) printLogs() {
	_, err := fmt.Println("--- LOGS FROM FAILED TEST:")
	assert.NoError(l.t, err, "Failed to write to stdout")

	l.lck.Lock()
	defer l.lck.Unlock()
	testNames := funk.Keys(l.pendingLogs)
	slices.Sort(testNames)

	for _, testName := range testNames {
		prefix := "    "
		if len(testNames) > 1 {
			prefix = fmt.Sprintf("    [%s] ", testName)
		}

		for _, pendingLog := range l.pendingLogs[testName] {
			fieldsJson, err := json.MarshalIndent(pendingLog.fields, "", "    ")
			assert.NoError(l.t, err, "failed to marshal logger fields as JSON")

			_, err = fmt.Printf(
				"%s%s %s: %s (channel = %s, fields = %s)\n",
				prefix,
				pendingLog.timestamp.Format("2006-01-02 15:04:05.999Z07:00"),
				pendingLog.level,
				pendingLog.message,
				pendingLog.channel,
				string(fieldsJson),
			)
			assert.NoError(l.t, err, "Failed to write to stdout")
		}
	}

	_, err = fmt.Println("--- END OF LOGS")
	assert.NoError(l.t, err, "Failed to write to stdout")
}
