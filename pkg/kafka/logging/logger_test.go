package logging_test

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/kafka/logging"
	"github.com/justtrackio/gosoline/pkg/log"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/testify/suite"
	"github.com/twmb/franz-go/pkg/kgo"
)

type LoggerTestSuite struct {
	suite.Suite
	baseLogger *logMocks.Logger
	logger     kgo.Logger
}

func TestLoggerTestSuite(t *testing.T) {
	suite.Run(t, new(LoggerTestSuite))
}

func (s *LoggerTestSuite) SetupTest() {
	s.baseLogger = logMocks.NewLogger(s.T())
	s.baseLogger.EXPECT().WithChannel("stream.kafka").Return(s.baseLogger).Once()
	s.logger = logging.NewKafkaLogger(context.Background(), s.baseLogger)
}

func (s *LoggerTestSuite) TestError() {
	s.baseLogger.EXPECT().WithFields(log.Fields{}).Return(s.baseLogger).Once()
	s.baseLogger.EXPECT().Error(matcher.Context, "some error")

	s.logger.Log(kgo.LogLevelError, "some error")
}

func (s *LoggerTestSuite) TestWarn() {
	s.baseLogger.EXPECT().WithFields(log.Fields{}).Return(s.baseLogger).Once()
	s.baseLogger.EXPECT().Warn(matcher.Context, "some error")

	s.logger.Log(kgo.LogLevelWarn, "some error")
}

func (s *LoggerTestSuite) TestInfo() {
	s.baseLogger.EXPECT().WithFields(log.Fields{}).Return(s.baseLogger).Once()
	s.baseLogger.EXPECT().Info(matcher.Context, "some error")

	s.logger.Log(kgo.LogLevelInfo, "some error")
}

func (s *LoggerTestSuite) TestDebug() {
	s.baseLogger.EXPECT().WithFields(log.Fields{}).Return(s.baseLogger).Once()
	s.baseLogger.EXPECT().Debug(matcher.Context, "some error")

	s.logger.Log(kgo.LogLevelDebug, "some error")
}

func (s *LoggerTestSuite) TestFields() {
	s.baseLogger.EXPECT().WithFields(log.Fields{
		"field_1": 1,
		"field_2": "2",
		"field_3": 3.0,
	}).Return(s.baseLogger).Once()
	s.baseLogger.EXPECT().Error(matcher.Context, "some error")

	s.logger.Log(kgo.LogLevelError, "some error", "field_1", 1, "field_2", "2", "field_3", 3.0)
}

func (s *LoggerTestSuite) TestFields_DropTrailingFieldWithoutValue() {
	s.baseLogger.EXPECT().WithFields(log.Fields{
		"field_1": 1,
		"field_2": "2",
		"field_3": 3.0,
	}).Return(s.baseLogger).Once()
	s.baseLogger.EXPECT().Error(matcher.Context, "some error")

	s.logger.Log(kgo.LogLevelError, "some error", "field_1", 1, "field_2", "2", "field_3", 3.0, "field_4")
}

func (s *LoggerTestSuite) TestFields_OneKeyNoValue() {
	s.baseLogger.EXPECT().WithFields(log.Fields{}).Return(s.baseLogger).Once()
	s.baseLogger.EXPECT().Error(matcher.Context, "some error")

	s.logger.Log(kgo.LogLevelError, "some error", "field_1")
}
