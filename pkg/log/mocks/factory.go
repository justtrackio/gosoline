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
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/objx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type Mock interface {
	String() string
	TestData() objx.Map
	Test(t mock.TestingT)
	On(methodName string, arguments ...any) *mock.Call
	Called(arguments ...any) mock.Arguments
	MethodCalled(methodName string, arguments ...any) mock.Arguments
	AssertExpectations(t mock.TestingT) bool
	AssertNumberOfCalls(t mock.TestingT, methodName string, expectedCalls int) bool
	AssertCalled(t mock.TestingT, methodName string, arguments ...any) bool
	AssertNotCalled(t mock.TestingT, methodName string, arguments ...any) bool
	IsMethodCallable(t mock.TestingT, methodName string, arguments ...any) bool
}

type LoggerMock interface {
	log.Logger
	Mock
	EXPECT() *Logger_Expecter
}

type GosoLoggerMock interface {
	log.GosoLogger
	Mock
	EXPECT() *GosoLogger_Expecter
}

type loggerMockOptions struct {
	t              *testing.T
	mockUntilLevel *int
}

type LoggerMockOption func(*loggerMockOptions)

type loggerMockState struct {
	t              *testing.T
	currentChannel string
	currentFields  log.Fields
	lck            *sync.Mutex
	pendingLogs    map[string][]pendingLogMessage
}

type loggerMock struct {
	*Logger
	*loggerMockState
}

type gosoLoggerMock struct {
	*GosoLogger
	*loggerMockState
}

func (l *gosoLoggerMock) Option(options ...log.Option) error {
	if hasExpectedCall(l.ExpectedCalls, "Option") {
		return l.GosoLogger.Option(options...)
	}

	return nil
}

func (l *gosoLoggerMock) Close(ctx context.Context) error {
	if hasExpectedCall(l.ExpectedCalls, "Close") {
		return l.GosoLogger.Close(ctx)
	}

	return nil
}

type pendingLogMessage struct {
	message   string
	level     string
	channel   string
	fields    log.Fields
	timestamp time.Time
}

func newLoggerMockState(t *testing.T) *loggerMockState {
	state := &loggerMockState{
		t:              t,
		currentChannel: "main",
		currentFields:  log.Fields{},
		lck:            &sync.Mutex{},
		pendingLogs:    map[string][]pendingLogMessage{},
	}

	if state.t != nil {
		state.t.Cleanup(func() {
			if !state.t.Failed() {
				return
			}

			state.printLogs()
		})
	}

	return state
}

func (l *loggerMockState) withChannel(channel string) *loggerMockState {
	return &loggerMockState{
		t:              l.t,
		currentChannel: channel,
		currentFields:  l.currentFields,
		lck:            l.lck,
		pendingLogs:    l.pendingLogs,
	}
}

func (l *loggerMockState) withFields(fields log.Fields) *loggerMockState {
	return &loggerMockState{
		t:              l.t,
		currentChannel: l.currentChannel,
		currentFields:  funk.MergeMaps(l.currentFields, fields),
		lck:            l.lck,
		pendingLogs:    l.pendingLogs,
	}
}

func hasExpectedCall(expectedCalls []*mock.Call, method string) bool {
	_, ok := funk.FindFirstFunc(expectedCalls, func(call *mock.Call) bool {
		return call.Method == method
	})

	return ok
}

func (l *loggerMock) WithChannel(channel string) log.Logger {
	// forward potential calls to the underlying mock if we expect some
	if hasExpectedCall(l.ExpectedCalls, "WithChannel") {
		l.Logger.WithChannel(channel)
	}

	return &loggerMock{
		Logger:          l.Logger,
		loggerMockState: l.withChannel(channel),
	}
}

func (l *loggerMock) WithFields(fields log.Fields) log.Logger {
	// forward potential calls to the underlying mock if we expect some
	if hasExpectedCall(l.ExpectedCalls, "WithFields") {
		l.Logger.WithFields(fields)
	}

	return &loggerMock{
		Logger:          l.Logger,
		loggerMockState: l.withFields(fields),
	}
}

func (l *gosoLoggerMock) WithChannel(channel string) log.Logger {
	// forward potential calls to the underlying mock if we expect some
	if hasExpectedCall(l.ExpectedCalls, "WithChannel") {
		l.GosoLogger.WithChannel(channel)
	}

	return &gosoLoggerMock{
		GosoLogger:      l.GosoLogger,
		loggerMockState: l.withChannel(channel),
	}
}

func (l *gosoLoggerMock) WithFields(fields log.Fields) log.Logger {
	// forward potential calls to the underlying mock if we expect some
	if hasExpectedCall(l.ExpectedCalls, "WithFields") {
		l.GosoLogger.WithFields(fields)
	}

	return &gosoLoggerMock{
		GosoLogger:      l.GosoLogger,
		loggerMockState: l.withFields(fields),
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
		Logger:          baseLogger,
		loggerMockState: newLoggerMockState(options.t),
	}

	if options.mockUntilLevel != nil {
		logger.mockLoggerMethod(logger.On, logger, "Debug", log.LevelDebug, *options.mockUntilLevel >= log.PriorityDebug)
		logger.mockLoggerMethod(logger.On, logger, "Info", log.LevelInfo, *options.mockUntilLevel >= log.PriorityInfo)
		logger.mockLoggerMethod(logger.On, logger, "Warn", log.LevelWarn, *options.mockUntilLevel >= log.PriorityWarn)
		logger.mockLoggerMethod(logger.On, logger, "Error", log.LevelError, *options.mockUntilLevel >= log.PriorityError)
	}

	return logger
}

// NewGosoLoggerMock creates a new GosoLogger mock with the given options.
func NewGosoLoggerMock(opts ...LoggerMockOption) GosoLoggerMock {
	var options loggerMockOptions
	for _, opt := range opts {
		opt(&options)
	}

	var baseLogger *GosoLogger
	if options.t != nil {
		baseLogger = NewGosoLogger(options.t)
	} else {
		baseLogger = new(GosoLogger)
	}

	logger := &gosoLoggerMock{
		GosoLogger:      baseLogger,
		loggerMockState: newLoggerMockState(options.t),
	}

	if options.mockUntilLevel != nil {
		logger.mockLoggerMethod(logger.On, logger, "Debug", log.LevelDebug, *options.mockUntilLevel >= log.PriorityDebug)
		logger.mockLoggerMethod(logger.On, logger, "Info", log.LevelInfo, *options.mockUntilLevel >= log.PriorityInfo)
		logger.mockLoggerMethod(logger.On, logger, "Warn", log.LevelWarn, *options.mockUntilLevel >= log.PriorityWarn)
		logger.mockLoggerMethod(logger.On, logger, "Error", log.LevelError, *options.mockUntilLevel >= log.PriorityError)
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

func (l *loggerMockState) mockLoggerMethod(on func(string, ...any) *mock.Call, returnValue any, method string, level string, allowed bool) {
	anythings := make(mock.Arguments, 0)
	f := l.inspectLogFunction(level, allowed)

	for i := 0; i < 10; i++ {
		anythings = append(anythings, mock.Anything)
		anythingsWithCtx := append([]any{matcher.Context}, anythings...)
		on(method, anythingsWithCtx...).Run(f).Return(returnValue).Maybe()
	}
}

func (l *loggerMockState) inspectLogFunction(level string, allowed bool) func(args mock.Arguments) {
	return func(args mock.Arguments) {
		msg := args.Get(1).(string)
		msg = fmt.Sprintf(msg, args[2:]...)

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

func (l *loggerMockState) printLogs() {
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
