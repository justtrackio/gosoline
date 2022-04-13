package logging_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/kafka/logging"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
)

func TestKafkaLogger(t *testing.T) {
	var (
		logger            = new(logMocks.Logger)
		loggerWithChannel = new(logMocks.Logger)
	)
	defer logger.AssertExpectations(t)
	defer loggerWithChannel.AssertExpectations(t)

	logger.On("WithChannel", "stream.kafka").Return(loggerWithChannel).Once()

	loggerWithChannel.On("Debug", "debug message").Once()
	loggerWithChannel.On("Error", "error message").Once()

	kLogger := logging.NewKafkaLogger(logger)
	kLogger.DebugLogger().Printf("debug message")
	kLogger.ErrorLogger().Printf("error message")
}
