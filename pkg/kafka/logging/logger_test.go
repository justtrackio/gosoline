package logging_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/kafka/logging"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
)

func TestKafkaLogger(t *testing.T) {
	var (
		logger            = logMocks.NewLogger(t)
		loggerWithChannel = logMocks.NewLogger(t)
	)

	logger.EXPECT().WithChannel("stream.kafka").Return(loggerWithChannel).Once()

	loggerWithChannel.EXPECT().Debug("debug message").Once()
	loggerWithChannel.EXPECT().Error("error message").Once()
	loggerWithChannel.EXPECT().Info("not the leader").Once()
	loggerWithChannel.EXPECT().Info("Not Leader For Partition: the client attempted to send messages to a replica that is not the leader for some partition, the client's metadata are likely out of date").Once()

	kLogger := logging.NewKafkaLogger(logger)
	kLogger.DebugLogger().Printf("debug message")
	kLogger.ErrorLogger().Printf("error message")
	kLogger.ErrorLogger().Printf("not the leader")
	kLogger.ErrorLogger().Printf("Not Leader For Partition: the client attempted to send messages to a replica that is not the leader for some partition, the client's metadata are likely out of date")
}
