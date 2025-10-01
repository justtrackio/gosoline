package logging_test

import (
	"errors"
	"testing"

	"github.com/justtrackio/gosoline/pkg/kafka/logging"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
)

func TestKafkaLogger(t *testing.T) {
	var (
		logger           = logMocks.NewLoggerMock(logMocks.WithTestingT(t))
		nonCriticalError = errors.New("Not Leader For Partition")
	)

	logger.EXPECT().Debug(matcher.Context, "debug message").Once()
	logger.EXPECT().Error(matcher.Context, "error message").Once()
	logger.EXPECT().Info(matcher.Context, "not the leader").Once()
	logger.EXPECT().Info(matcher.Context, "error: %s", nonCriticalError).Once()
	logger.EXPECT().Info(matcher.Context, "unexpected EOF").Once()

	kLogger := logging.NewKafkaLogger(logger)
	kLogger.DebugLogger().Printf("debug message")
	kLogger.ErrorLogger().Printf("error message")
	kLogger.ErrorLogger().Printf("not the leader")
	kLogger.ErrorLogger().Printf("error: %s", nonCriticalError)
	kLogger.ErrorLogger().Printf("unexpected EOF")
}
