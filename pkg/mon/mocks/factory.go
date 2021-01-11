package mocks

import (
	"fmt"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/stretchr/testify/mock"
)

func NewLoggerMock() *Logger {
	logger := new(Logger)

	logger.On("WithChannel", mock.AnythingOfType("string")).Return(logger).Maybe()
	logger.On("WithContext", mock.Anything).Return(logger).Maybe()
	logger.On("WithFields", mock.AnythingOfType("mon.Fields")).Return(logger).Maybe()

	return logger
}

func NewLoggerMockedAll() *Logger {
	return NewLoggerMockedUntilLevel(mon.Panic)
}

// return a logger mocked up to the given log level. All other calls will cause an error and fail the test.
func NewLoggerMockedUntilLevel(level string) *Logger {
	logger := NewLoggerMock()

	levelMap := map[string]int{
		mon.Trace: 0,
		mon.Debug: 1,
		mon.Info:  2,
		mon.Warn:  3,
		mon.Error: 4,
		mon.Fatal: 5,
		mon.Panic: 6,
	}

	levelIndex, ok := levelMap[level]

	if !ok {
		panic(fmt.Errorf("failed to find level to mock: %s", level))
	}

	mockLoggerMethod(logger, "Debug", "Debugf", mon.Debug, levelIndex >= levelMap[mon.Debug])
	mockLoggerMethod(logger, "Info", "Infof", mon.Info, levelIndex >= levelMap[mon.Info])
	mockLoggerMethod(logger, "Warn", "Warnf", mon.Warn, levelIndex >= levelMap[mon.Warn])
	mockLoggerMethod(logger, "Error", "Errorf", mon.Error, levelIndex >= levelMap[mon.Error])
	mockLoggerMethod(logger, "Fatal", "Fatalf", mon.Fatal, levelIndex >= levelMap[mon.Fatal])
	mockLoggerMethod(logger, "Panic", "Panicf", mon.Panic, levelIndex >= levelMap[mon.Panic])

	return logger
}

func inspectLogFunction(level string, withFormat bool, allowed bool) func(args mock.Arguments) {
	return func(args mock.Arguments) {
		if level == mon.Panic || level == mon.Fatal {
			err := args.Get(0).(error)
			var msg string
			if withFormat {
				msg = fmt.Sprintf(args.Get(1).(string), args[2:]...)
			} else {
				msg = fmt.Sprint(args[1:])
			}
			// we have to stop the test, these methods never return
			panic(fmt.Errorf("panic or fatal logging method called with error '%w' and message '%s'", err, msg))
		}

		// ensure we use formatting markers exactly when using a method with formatting
		var msg string
		if level == mon.Error {
			msg = fmt.Sprint(args.Get(1))
		} else if len(args) > 0 {
			msg = fmt.Sprint(args.Get(0))
		}

		if !allowed {
			panic(fmt.Errorf("invalid log message '%s'. Logs of level %s are not allowed", msg, level))
		}
	}
}

func mockLoggerMethod(logger *Logger, method string, methodWithFormat string, level string, allowed bool) {
	anythings := make(mock.Arguments, 0)
	f := inspectLogFunction(level, false, allowed)
	fWithFormat := inspectLogFunction(level, true, allowed)

	for i := 0; i < 10; i++ {
		anythings = append(anythings, mock.Anything)
		logger.On(method, anythings...).Run(f).Return(logger).Maybe()
		logger.On(methodWithFormat, anythings...).Run(fWithFormat).Return(logger).Maybe()
	}
}

func NewMetricWriterMockedAll() *MetricWriter {
	mw := new(MetricWriter)
	mw.On("GetPriority").Return(mon.PriorityLow).Maybe()
	mw.On("Write", mock.AnythingOfType("mon.MetricData")).Return().Maybe()
	mw.On("WriteOne", mock.AnythingOfType("*mon.MetricDatum")).Return().Maybe()

	return mw
}
