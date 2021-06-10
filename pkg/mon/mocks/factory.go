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
	return NewLoggerMockedUntilLevel(mon.Error)
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
	}

	levelIndex, ok := levelMap[level]

	if !ok {
		panic(fmt.Errorf("failed to find level to mock: %s", level))
	}

	mockLoggerMethod(logger, "Debug", mon.Debug, levelIndex >= levelMap[mon.Debug])
	mockLoggerMethod(logger, "Info", mon.Info, levelIndex >= levelMap[mon.Info])
	mockLoggerMethod(logger, "Warn", mon.Warn, levelIndex >= levelMap[mon.Warn])
	mockLoggerMethod(logger, "Error", mon.Error, levelIndex >= levelMap[mon.Error])

	return logger
}

func inspectLogFunction(level string, allowed bool) func(args mock.Arguments) {
	return func(args mock.Arguments) {
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

func mockLoggerMethod(logger *Logger, method string, level string, allowed bool) {
	anythings := make(mock.Arguments, 0)
	f := inspectLogFunction(level, allowed)

	for i := 0; i < 10; i++ {
		anythings = append(anythings, mock.Anything)
		logger.On(method, anythings...).Run(f).Return(logger).Maybe()
	}
}

func NewMetricWriterMockedAll() *MetricWriter {
	mw := new(MetricWriter)
	mw.On("GetPriority").Return(mon.PriorityLow).Maybe()
	mw.On("Write", mock.AnythingOfType("mon.MetricData")).Return().Maybe()
	mw.On("WriteOne", mock.AnythingOfType("*mon.MetricDatum")).Return().Maybe()

	return mw
}
