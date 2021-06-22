package log_test

import (
	"context"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"testing"
)

type LoggerMessageEncodeHandlerTestSuite struct {
	suite.Suite

	logger  *mocks.Logger
	encoder *log.MessageWithLoggingFieldsEncoder
}

func (s *LoggerMessageEncodeHandlerTestSuite) SetupTest() {
	s.logger = new(mocks.Logger)
	s.encoder = log.NewMessageWithLoggingFieldsEncoderWithInterfaces(s.logger)
}

func (s *LoggerMessageEncodeHandlerTestSuite) TestEncodeEmpty() {
	ctx := context.Background()
	attributes := make(map[string]interface{})

	_, attributes, err := s.encoder.Encode(ctx, nil, attributes)

	s.NoError(err)
	s.Empty(attributes)
}

func (s *LoggerMessageEncodeHandlerTestSuite) TestEncodeSuccess() {
	s.logger.On("Warn", "omitting logger context field %s of type %T during message encoding", "fieldC", mock.Anything)

	ctx := context.Background()
	ctx = log.AppendLoggerContextField(ctx, map[string]interface{}{
		"fieldA": "text",
		"fieldB": 1,
		"fieldC": struct{}{},
	})

	attributes := make(map[string]interface{})
	_, attributes, err := s.encoder.Encode(ctx, nil, attributes)

	s.NoError(err)
	s.Len(attributes, 1)
	s.Contains(attributes, log.MessageAttributeLoggerContext)
	s.JSONEq(`{"fieldA":"text","fieldB":1}`, attributes[log.MessageAttributeLoggerContext].(string))

	s.logger.AssertExpectations(s.T())
}

func (s *LoggerMessageEncodeHandlerTestSuite) TestDecodeEmpty() {
	ctx := context.Background()
	attributes := map[string]interface{}{}

	_, _, err := s.encoder.Decode(ctx, nil, attributes)

	s.NoError(err)
}

func (s *LoggerMessageEncodeHandlerTestSuite) TestDecodeTypeError() {
	s.logger.On("Warn", "encoded logger context fields should be of type string but instead are of type %T", 1)

	ctx := context.Background()
	attributes := map[string]interface{}{
		log.MessageAttributeLoggerContext: 1,
	}

	_, _, err := s.encoder.Decode(ctx, nil, attributes)

	s.NoError(err)
	s.logger.AssertExpectations(s.T())
}

func (s *LoggerMessageEncodeHandlerTestSuite) TestDecodeJsonError() {
	s.logger.On("Warn", "can not json unmarshal logger context fields during message decoding")

	ctx := context.Background()
	attributes := map[string]interface{}{
		log.MessageAttributeLoggerContext: `broken`,
	}

	_, _, err := s.encoder.Decode(ctx, nil, attributes)

	s.NoError(err)
	s.logger.AssertExpectations(s.T())
}

func (s *LoggerMessageEncodeHandlerTestSuite) TestDecodeSuccess() {
	ctx := context.Background()
	attributes := map[string]interface{}{
		log.MessageAttributeLoggerContext: `{"fieldA":"text","fieldB":1}`,
	}

	ctx, attributes, err := s.encoder.Decode(ctx, nil, attributes)

	s.NoError(err)
	s.NotContains(attributes, log.MessageAttributeLoggerContext)

	fields := log.ContextLoggerFieldsResolver(ctx)
	s.Contains(fields, "fieldA")
	s.Equal("text", fields["fieldA"])
	s.Contains(fields, "fieldB")
	s.Equal(1.0, fields["fieldB"])
}

func TestLoggerMessageEncodeHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(LoggerMessageEncodeHandlerTestSuite))
}
