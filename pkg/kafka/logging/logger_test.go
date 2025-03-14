package logging_test

import (
	"errors"
	"testing"

	"github.com/justtrackio/gosoline/pkg/kafka/logging"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
)

func TestKafkaLogger(t *testing.T) {
	var (
		logger            = logMocks.NewLogger(t)
		loggerWithChannel = logMocks.NewLogger(t)
		nonCriticalError  = errors.New("Not Leader For Partition")
	)

	logger.EXPECT().WithChannel("stream.kafka").Return(loggerWithChannel).Once()

	loggerWithChannel.EXPECT().Debug("debug message").Once()
	loggerWithChannel.EXPECT().Error("error message").Once()
	loggerWithChannel.EXPECT().Info("not the leader").Once()
	loggerWithChannel.EXPECT().Info("error: %s", nonCriticalError).Once()

	kLogger := logging.NewKafkaLogger(logger)
	kLogger.DebugLogger().Printf("debug message")
	kLogger.ErrorLogger().Printf("error message")
	kLogger.ErrorLogger().Printf("not the leader")
	kLogger.ErrorLogger().Printf("error: %s", nonCriticalError)
}
