package log_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

func TestLoggerMessageEncodeHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(LoggerMessageEncodeHandlerTestSuite))
}

type LoggerMessageEncodeHandlerTestSuite struct {
	suite.Suite

	logger  *mocks.Logger
	encoder *log.MessageWithLoggingFieldsEncoder
}

func (s *LoggerMessageEncodeHandlerTestSuite) SetupTest() {
	s.logger = mocks.NewLogger(s.T())
	s.encoder = log.NewMessageWithLoggingFieldsEncoderWithInterfaces(s.logger)
}

func (s *LoggerMessageEncodeHandlerTestSuite) TestEncodeEmpty() {
	ctx := s.T().Context()
	attributes := make(map[string]string)

	_, attributes, err := s.encoder.Encode(ctx, nil, attributes)

	s.NoError(err)
	s.Empty(attributes)
}

func (s *LoggerMessageEncodeHandlerTestSuite) TestEncodeSuccess() {
	s.logger.EXPECT().Warn("omitting logger context field %s of type %T during message encoding", "fieldC", mock.Anything)

	ctx := s.T().Context()
	ctx = log.AppendGlobalContextFields(ctx, map[string]any{
		"fieldA": "text",
		"fieldB": "1",
		"fieldC": struct{}{},
	})

	attributes := make(map[string]string)
	_, attributes, err := s.encoder.Encode(ctx, nil, attributes)

	s.NoError(err)
	s.Len(attributes, 1)
	s.Contains(attributes, log.MessageAttributeLoggerContext)
	s.JSONEq(`{"fieldA":"text","fieldB":"1"}`, attributes[log.MessageAttributeLoggerContext])
}

func (s *LoggerMessageEncodeHandlerTestSuite) TestDecodeEmpty() {
	ctx := s.T().Context()
	attributes := map[string]string{}

	_, _, err := s.encoder.Decode(ctx, nil, attributes)

	s.NoError(err)
}

func (s *LoggerMessageEncodeHandlerTestSuite) TestDecodeJsonError() {
	s.logger.EXPECT().Warn("can not json unmarshal logger context fields during message decoding")

	ctx := s.T().Context()
	attributes := map[string]string{
		log.MessageAttributeLoggerContext: `broken`,
	}

	_, _, err := s.encoder.Decode(ctx, nil, attributes)

	s.NoError(err)
}

func (s *LoggerMessageEncodeHandlerTestSuite) TestDecodeSuccess() {
	ctx := s.T().Context()
	attributes := map[string]string{
		log.MessageAttributeLoggerContext: `{"fieldA":"text","fieldB":1}`,
	}

	ctx, attributes, err := s.encoder.Decode(ctx, nil, attributes)

	s.NoError(err)
	s.NotContains(attributes, log.MessageAttributeLoggerContext)

	fields := log.GlobalContextFieldsResolver(ctx)
	s.Contains(fields, "fieldA")
	s.Equal("text", fields["fieldA"])
	s.Contains(fields, "fieldB")
	s.Equal(1.0, fields["fieldB"])
}
