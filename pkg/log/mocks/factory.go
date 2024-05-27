package mocks

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/stretchr/testify/mock"
)

func NewLoggerMock() *Logger {
	logger := new(Logger)

	logger.EXPECT().WithChannel(mock.AnythingOfType("string")).Return(logger).Maybe()
	logger.EXPECT().WithContext(mock.Anything).Return(logger).Maybe()
	logger.EXPECT().WithFields(mock.Anything).Return(logger).Maybe()

	return logger
}

func NewLoggerMockedAll() *Logger {
	return NewLoggerMockedUntilLevel(log.PriorityError)
}

// NewLoggerMockedUntilLevel returns a logger mocked up to the given log level. All other calls will cause an error and fail the test.
func NewLoggerMockedUntilLevel(level int) *Logger {
	logger := NewLoggerMock()

	mockLoggerMethod(logger.EXPECT().Debug, log.LevelDebug, level >= log.PriorityDebug)
	mockLoggerMethod(logger.EXPECT().Info, log.LevelInfo, level >= log.PriorityInfo)
	mockLoggerMethod(logger.EXPECT().Warn, log.LevelWarn, level >= log.PriorityWarn)
	mockLoggerMethod(logger.EXPECT().Error, log.LevelError, level >= log.PriorityError)

	return logger
}

type call[C call2[C]] interface {
	call2[C]
	Run(run func(format string, args ...interface{})) C
}

type call2[C call3] interface {
	call3
	Return() C
}

type call3 interface {
	Maybe() *mock.Call
}

func mockLoggerMethod[C call[C]](method func(format interface{}, args ...interface{}) C, level string, allowed bool) {
	anythings := make(mock.Arguments, 0)
	f := inspectLogFunction(level, allowed)

	for i := 0; i < 10; i++ {
		method(mock.AnythingOfType("string"), anythings...).Run(f).Return().Maybe()
		anythings = append(anythings, mock.Anything)
	}
}

func inspectLogFunction(level string, allowed bool) func(format string, args ...interface{}) {
	return func(format string, args ...interface{}) {
		if !allowed {
			panic(fmt.Errorf("invalid log message '%s' and parameters %v. Logs of level %s are not allowed", format, args, level))
		}
	}
}
