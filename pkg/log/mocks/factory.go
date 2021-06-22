package mocks

import (
	"fmt"
	"github.com/applike/gosoline/pkg/log"
	"github.com/stretchr/testify/mock"
)

func NewLoggerMock() *Logger {
	logger := new(Logger)

	logger.On("WithChannel", mock.AnythingOfType("string")).Return(logger).Maybe()
	logger.On("WithContext", mock.Anything).Return(logger).Maybe()
	logger.On("WithFields", mock.Anything).Return(logger).Maybe()

	return logger
}

func NewLoggerMockedAll() *Logger {
	return NewLoggerMockedUntilLevel(log.PriorityError)
}

// return a logger mocked up to the given log level. All other calls will cause an error and fail the test.
func NewLoggerMockedUntilLevel(level int) *Logger {
	logger := NewLoggerMock()

	mockLoggerMethod(logger, "Debug", log.LevelDebug, level >= log.PriorityDebug)
	mockLoggerMethod(logger, "Info", log.LevelInfo, level >= log.PriorityInfo)
	mockLoggerMethod(logger, "Warn", log.LevelWarn, level >= log.PriorityWarn)
	mockLoggerMethod(logger, "Error", log.LevelError, level >= log.PriorityError)

	return logger
}

func mockLoggerMethod(logger *Logger, method string, level string, allowed bool) {
	anythings := make(mock.Arguments, 0)
	f := inspectLogFunction(level, allowed)

	for i := 0; i < 10; i++ {
		anythings = append(anythings, mock.Anything)
		logger.On(method, anythings...).Run(f).Return(logger).Maybe()
	}
}

func inspectLogFunction(level string, allowed bool) func(args mock.Arguments) {
	return func(args mock.Arguments) {
		if !allowed {
			panic(fmt.Errorf("invalid log message '%s'. Logs of level %s are not allowed", args.Get(0), level))
		}
	}
}
