//go:build integration

package retry_handler_sqs

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/encoding/base64"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/test/suite"
	"google.golang.org/protobuf/proto"
)

func TestRetryHandlerSqsTestSuite(t *testing.T) {
	suite.Run(t, new(RetryHandlerSqsTestSuite))
}

type RetryHandlerSqsTestSuite struct {
	suite.Suite
	callback  *Callback
	sqsOutput stream.Output
}

func (s *RetryHandlerSqsTestSuite) SetupSuite() []suite.Option {
	s.callback = NewCallback()
	// lock until we set up app under test and mode
	s.callback.lck.Lock()

	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithConfigFile("config.dist.yml"),
		suite.WithConsumer(func(ctx context.Context, config cfg.Config, logger log.Logger) (stream.ConsumerCallback[DataModel], error) {
			sqsOutput, err := stream.NewSqsOutput(ctx, config, logger, &stream.SqsOutputSettings{
				QueueId:           "test",
				ClientName:        "default",
				VisibilityTimeout: 5,
			})
			if err != nil {
				return nil, err
			}

			s.sqsOutput = sqsOutput

			return s.callback, nil
		}),
		suite.WithSharedEnvironment(),
	}
}

func (s *RetryHandlerSqsTestSuite) TestRetryBatch(aut suite.AppUnderTest) {
	s.setupCallback(aut, 2, 4)
	defer s.callback.lck.Lock()

	s.writeAggregateMessage()

	aut.WaitDone()

	s.Len(s.callback.receivedModels, 4, "the model should have been received 4 times")
	s.Equal(s.callback.receivedModels[0], DataModel{Id: 1}, "the first receive should have the correct body")
	s.Equal(s.callback.receivedModels[1], DataModel{Id: 2}, "the second receive should have the correct body")
	s.Equal(s.callback.receivedModels[2], DataModel{Id: 1}, "the first retry receive should have the correct body")
	s.Equal(s.callback.receivedModels[3], DataModel{Id: 2}, "the second retry receive should have the correct body")

	s.Equal(s.callback.receivedAttributes[0], map[string]string{
		stream.AttributeEncoding: stream.EncodingJson.String(),
	}, "the first receive should have correct attributes")
	s.Equal(s.callback.receivedAttributes[1], map[string]string{
		stream.AttributeEncoding: stream.EncodingJson.String(),
	}, "the second receive should have correct attributes")
	s.Equal(s.callback.receivedAttributes[2], map[string]string{
		stream.AttributeEncoding: stream.EncodingJson.String(),
		stream.AttributeRetry:    "true",
		stream.AttributeRetryId:  s.callback.receivedAttributes[2][stream.AttributeRetryId],
		"goso.retry.sqs":         "true",
	}, "the first retry receive should have the retry attribute")
	s.Equal(s.callback.receivedAttributes[3], map[string]string{
		stream.AttributeEncoding: stream.EncodingJson.String(),
		stream.AttributeRetry:    "true",
		stream.AttributeRetryId:  s.callback.receivedAttributes[3][stream.AttributeRetryId],
		"goso.retry.sqs":         "true",
	}, "the second retry receive should have the retry attribute")
}

func (s *RetryHandlerSqsTestSuite) TestRetryBatchOfCompressedMessages(aut suite.AppUnderTest) {
	s.setupCallback(aut, 2, 4)
	defer s.callback.lck.Lock()

	s.writeAggregateCompressedMessage()

	aut.WaitDone()

	s.Len(s.callback.receivedModels, 4, "the model should have been received 4 times")
	s.Equal(s.callback.receivedModels[0], DataModel{Id: 1}, "the first receive should have the correct body")
	s.Equal(s.callback.receivedModels[1], DataModel{Id: 2}, "the second receive should have the correct body")
	s.Equal(s.callback.receivedModels[2], DataModel{Id: 1}, "the first retry receive should have the correct body")
	s.Equal(s.callback.receivedModels[3], DataModel{Id: 2}, "the second retry receive should have the correct body")

	s.Equal(s.callback.receivedAttributes[0], map[string]string{
		stream.AttributeEncoding:    stream.EncodingJson.String(),
		stream.AttributeCompression: stream.CompressionGZip.String(),
	}, "the first receive should have correct attributes")
	s.Equal(s.callback.receivedAttributes[1], map[string]string{
		stream.AttributeEncoding:    stream.EncodingJson.String(),
		stream.AttributeCompression: stream.CompressionGZip.String(),
	}, "the second receive should have correct attributes")
	s.Equal(s.callback.receivedAttributes[2], map[string]string{
		stream.AttributeEncoding:    stream.EncodingJson.String(),
		stream.AttributeCompression: stream.CompressionGZip.String(),
		stream.AttributeRetry:       "true",
		stream.AttributeRetryId:     s.callback.receivedAttributes[2][stream.AttributeRetryId],
		"goso.retry.sqs":            "true",
	}, "the first retry receive should have the retry attribute")
	s.Equal(s.callback.receivedAttributes[3], map[string]string{
		stream.AttributeEncoding:    stream.EncodingJson.String(),
		stream.AttributeCompression: stream.CompressionGZip.String(),
		stream.AttributeRetry:       "true",
		stream.AttributeRetryId:     s.callback.receivedAttributes[3][stream.AttributeRetryId],
		"goso.retry.sqs":            "true",
	}, "the second retry receive should have the retry attribute")
}

func (s *RetryHandlerSqsTestSuite) TestRetryBatchCompressed(aut suite.AppUnderTest) {
	s.setupCallback(aut, 2, 4)
	defer s.callback.lck.Lock()

	s.writeCompressedAggregateMessage()

	aut.WaitDone()

	s.Len(s.callback.receivedModels, 4, "the model should have been received 4 times")
	s.Equal(s.callback.receivedModels[0], DataModel{Id: 1}, "the first receive should have the correct body")
	s.Equal(s.callback.receivedModels[1], DataModel{Id: 2}, "the second receive should have the correct body")
	s.Equal(s.callback.receivedModels[2], DataModel{Id: 1}, "the first retry receive should have the correct body")
	s.Equal(s.callback.receivedModels[3], DataModel{Id: 2}, "the second retry receive should have the correct body")

	s.Equal(s.callback.receivedAttributes[0], map[string]string{
		stream.AttributeEncoding: stream.EncodingJson.String(),
	}, "the first receive should have correct attributes")
	s.Equal(s.callback.receivedAttributes[1], map[string]string{
		stream.AttributeEncoding: stream.EncodingJson.String(),
	}, "the second receive should have correct attributes")
	s.Equal(s.callback.receivedAttributes[2], map[string]string{
		stream.AttributeEncoding: stream.EncodingJson.String(),
		stream.AttributeRetry:    "true",
		stream.AttributeRetryId:  s.callback.receivedAttributes[2][stream.AttributeRetryId],
		"goso.retry.sqs":         "true",
	}, "the first retry receive should have the retry attribute")
	s.Equal(s.callback.receivedAttributes[3], map[string]string{
		stream.AttributeEncoding: stream.EncodingJson.String(),
		stream.AttributeRetry:    "true",
		stream.AttributeRetryId:  s.callback.receivedAttributes[3][stream.AttributeRetryId],
		"goso.retry.sqs":         "true",
	}, "the second retry receive should have the retry attribute")
}

func (s *RetryHandlerSqsTestSuite) TestRetryBatchWithProtobuf(aut suite.AppUnderTest) {
	s.setupCallback(aut, 2, 4)
	defer s.callback.lck.Lock()

	s.writeProtobufAggregateMessage()

	aut.WaitDone()

	s.Len(s.callback.receivedModels, 4, "the model should have been received 4 times")
	s.Equal(s.callback.receivedModels[0], DataModel{Id: 1}, "the first receive should have the correct body")
	s.Equal(s.callback.receivedModels[1], DataModel{Id: 2}, "the second receive should have the correct body")
	s.Equal(s.callback.receivedModels[2], DataModel{Id: 1}, "the first retry receive should have the correct body")
	s.Equal(s.callback.receivedModels[3], DataModel{Id: 2}, "the second retry receive should have the correct body")

	s.Equal(s.callback.receivedAttributes[0], map[string]string{
		stream.AttributeEncoding: stream.EncodingProtobuf.String(),
	}, "the first receive should have correct attributes")
	s.Equal(s.callback.receivedAttributes[1], map[string]string{
		stream.AttributeEncoding: stream.EncodingProtobuf.String(),
	}, "the second receive should have correct attributes")
	s.Equal(s.callback.receivedAttributes[2], map[string]string{
		stream.AttributeEncoding: stream.EncodingProtobuf.String(),
		stream.AttributeRetry:    "true",
		stream.AttributeRetryId:  s.callback.receivedAttributes[2][stream.AttributeRetryId],
		"goso.retry.sqs":         "true",
	}, "the first retry receive should have the retry attribute")
	s.Equal(s.callback.receivedAttributes[3], map[string]string{
		stream.AttributeEncoding: stream.EncodingProtobuf.String(),
		stream.AttributeRetry:    "true",
		stream.AttributeRetryId:  s.callback.receivedAttributes[3][stream.AttributeRetryId],
		"goso.retry.sqs":         "true",
	}, "the second retry receive should have the retry attribute")
}

func (s *RetryHandlerSqsTestSuite) TestRetryBatchWithProtobufAndCompression(aut suite.AppUnderTest) {
	s.setupCallback(aut, 2, 4)
	defer s.callback.lck.Lock()

	s.writeCompressedProtobufAggregateMessage()

	aut.WaitDone()

	s.Len(s.callback.receivedModels, 4, "the model should have been received 4 times")
	s.Equal(s.callback.receivedModels[0], DataModel{Id: 1}, "the first receive should have the correct body")
	s.Equal(s.callback.receivedModels[1], DataModel{Id: 2}, "the second receive should have the correct body")
	s.Equal(s.callback.receivedModels[2], DataModel{Id: 1}, "the first retry receive should have the correct body")
	s.Equal(s.callback.receivedModels[3], DataModel{Id: 2}, "the second retry receive should have the correct body")

	s.Equal(s.callback.receivedAttributes[0], map[string]string{
		stream.AttributeEncoding: stream.EncodingProtobuf.String(),
	}, "the first receive should have correct attributes")
	s.Equal(s.callback.receivedAttributes[1], map[string]string{
		stream.AttributeEncoding: stream.EncodingProtobuf.String(),
	}, "the second receive should have correct attributes")
	s.Equal(s.callback.receivedAttributes[2], map[string]string{
		stream.AttributeEncoding: stream.EncodingProtobuf.String(),
		stream.AttributeRetry:    "true",
		stream.AttributeRetryId:  s.callback.receivedAttributes[2][stream.AttributeRetryId],
		"goso.retry.sqs":         "true",
	}, "the first retry receive should have the retry attribute")
	s.Equal(s.callback.receivedAttributes[3], map[string]string{
		stream.AttributeEncoding: stream.EncodingProtobuf.String(),
		stream.AttributeRetry:    "true",
		stream.AttributeRetryId:  s.callback.receivedAttributes[3][stream.AttributeRetryId],
		"goso.retry.sqs":         "true",
	}, "the second retry receive should have the retry attribute")
}

func (s *RetryHandlerSqsTestSuite) TestRetrySingleMessage(aut suite.AppUnderTest) {
	s.setupCallback(aut, 1, 2)
	defer s.callback.lck.Lock()

	s.writeSingleMessage()

	aut.WaitDone()

	s.Len(s.callback.receivedModels, 2, "the model should have been received 2 times")
	s.Equal(s.callback.receivedModels[0], DataModel{Id: 1}, "the first receive should have the correct body")
	s.Equal(s.callback.receivedModels[1], DataModel{Id: 1}, "the first retry receive should have the correct body")

	s.Equal(s.callback.receivedAttributes[0], map[string]string{
		stream.AttributeEncoding: stream.EncodingJson.String(),
	}, "the first receive should have correct attributes")
	s.Equal(s.callback.receivedAttributes[1], map[string]string{
		stream.AttributeEncoding: stream.EncodingJson.String(),
	}, "the second receive should have correct attributes")
}

func (s *RetryHandlerSqsTestSuite) TestRetrySingleCompressedMessage(aut suite.AppUnderTest) {
	s.setupCallback(aut, 1, 2)
	defer s.callback.lck.Lock()

	s.writeSingleCompressedMessage()

	aut.WaitDone()

	s.Len(s.callback.receivedModels, 2, "the model should have been received 2 times")
	s.Equal(s.callback.receivedModels[0], DataModel{Id: 1}, "the first receive should have the correct body")
	s.Equal(s.callback.receivedModels[1], DataModel{Id: 1}, "the first retry receive should have the correct body")

	s.Equal(s.callback.receivedAttributes[0], map[string]string{
		stream.AttributeEncoding:    stream.EncodingJson.String(),
		stream.AttributeCompression: stream.CompressionGZip.String(),
	}, "the first receive should have correct attributes")
	s.Equal(s.callback.receivedAttributes[1], map[string]string{
		stream.AttributeEncoding:    stream.EncodingJson.String(),
		stream.AttributeCompression: stream.CompressionGZip.String(),
	}, "the second receive should have correct attributes")
}

func (s *RetryHandlerSqsTestSuite) writeAggregateMessage() {
	batch := []*stream.Message{
		stream.NewJsonMessage(`{ "Id": 1 }`),
		stream.NewJsonMessage(`{ "Id": 2 }`),
	}
	messages, err := json.Marshal(batch)
	s.NoError(err)

	msg := stream.NewJsonMessage(string(messages), map[string]string{
		stream.AttributeAggregate: strconv.FormatBool(true),
	})
	err = s.sqsOutput.WriteOne(context.Background(), msg)
	s.NoError(err)
}

func (s *RetryHandlerSqsTestSuite) writeAggregateCompressedMessage() {
	batch := []*stream.Message{
		stream.NewJsonMessage(s.compress([]byte(`{ "Id": 1 }`)), map[string]string{
			stream.AttributeCompression: stream.CompressionGZip.String(),
		}),
		stream.NewJsonMessage(s.compress([]byte(`{ "Id": 2 }`)), map[string]string{
			stream.AttributeCompression: stream.CompressionGZip.String(),
		}),
	}
	messages, err := json.Marshal(batch)
	s.NoError(err)

	msg := stream.NewJsonMessage(string(messages), map[string]string{
		stream.AttributeAggregate: strconv.FormatBool(true),
	})
	err = s.sqsOutput.WriteOne(context.Background(), msg)
	s.NoError(err)
}

func (s *RetryHandlerSqsTestSuite) writeCompressedAggregateMessage() {
	batch := []*stream.Message{
		stream.NewJsonMessage(`{ "Id": 1 }`),
		stream.NewJsonMessage(`{ "Id": 2 }`),
	}
	messages, err := json.Marshal(batch)
	s.NoError(err)

	body := s.compress(messages)

	msg := stream.NewJsonMessage(body, map[string]string{
		stream.AttributeAggregate:   strconv.FormatBool(true),
		stream.AttributeCompression: stream.CompressionGZip.String(),
	})
	err = s.sqsOutput.WriteOne(context.Background(), msg)
	s.NoError(err)
}

func (s *RetryHandlerSqsTestSuite) writeProtobufAggregateMessage() {
	messages := s.protobufBatch()

	msg := stream.NewJsonMessage(string(messages), map[string]string{
		stream.AttributeAggregate: strconv.FormatBool(true),
	})
	err := s.sqsOutput.WriteOne(context.Background(), msg)
	s.NoError(err)
}

func (s *RetryHandlerSqsTestSuite) writeCompressedProtobufAggregateMessage() {
	messages := s.protobufBatch()
	body := s.compress(messages)

	msg := stream.NewJsonMessage(body, map[string]string{
		stream.AttributeAggregate:   strconv.FormatBool(true),
		stream.AttributeCompression: stream.CompressionGZip.String(),
	})
	err := s.sqsOutput.WriteOne(context.Background(), msg)
	s.NoError(err)
}

func (s *RetryHandlerSqsTestSuite) protobufBatch() []byte {
	msg1, err := (&DataModel{Id: 1}).ToMessage()
	s.NoError(err)

	msg2, err := (&DataModel{Id: 2}).ToMessage()
	s.NoError(err)

	bytes1, err := proto.Marshal(msg1)
	s.NoError(err)

	bytes2, err := proto.Marshal(msg2)
	s.NoError(err)

	batch := []*stream.Message{
		stream.NewProtobufMessage(base64.EncodeToString(bytes1)),
		stream.NewProtobufMessage(base64.EncodeToString(bytes2)),
	}
	messages, err := json.Marshal(batch)
	s.NoError(err)

	return messages
}

func (s *RetryHandlerSqsTestSuite) writeSingleMessage() {
	msg := stream.NewJsonMessage(`{ "Id": 1 }`)
	err := s.sqsOutput.WriteOne(context.Background(), msg)
	s.NoError(err)
}

func (s *RetryHandlerSqsTestSuite) writeSingleCompressedMessage() {
	msg := stream.NewJsonMessage(s.compress([]byte(`{ "Id": 1 }`)), map[string]string{
		stream.AttributeCompression: stream.CompressionGZip.String(),
	})
	err := s.sqsOutput.WriteOne(context.Background(), msg)
	s.NoError(err)
}

func (s *RetryHandlerSqsTestSuite) setupCallback(aut suite.AppUnderTest, retryCount int, stopAt int) {
	s.callback.aut = aut
	s.callback.retryCount = retryCount
	s.callback.stopAt = stopAt
	s.callback.receivedAttributes = nil
	s.callback.receivedModels = nil
	s.callback.lck.Unlock()
}

func (s *RetryHandlerSqsTestSuite) compress(data []byte) string {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	_, err := w.Write(data)
	s.NoError(err)

	err = w.Close()
	s.NoError(err)

	return base64.EncodeToString(b.Bytes())
}
