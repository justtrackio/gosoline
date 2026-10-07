package stream_test

import (
	"context"
	"maps"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/cloud/aws/sqs"
	sqsMocks "github.com/justtrackio/gosoline/pkg/cloud/aws/sqs/mocks"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
)

func TestKafkaMessageSurvivesSqsRetry(t *testing.T) {
	record := kgo.Record{
		Value: []byte("message value"),
		Key:   []byte("record-key"),
		Headers: []kgo.RecordHeader{
			{Key: "text", Value: []byte("valid UTF-8")},
		},
	}
	message := stream.KafkaToGosoMessage(record)
	before := maps.Clone(message.Attributes)
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	queue := sqsMocks.NewQueue(t)
	queue.EXPECT().Send(t.Context(), mock.Anything).RunAndReturn(func(_ context.Context, sent *sqs.Message) error {
		var received stream.Message
		require.NoError(t, received.UnmarshalFromString(*sent.Body))
		restored, err := stream.NewKafkaMessage(&received)
		require.NoError(t, err)
		require.Equal(t, record.Value, restored.Value)
		require.Equal(t, record.Key, restored.Key)
		require.Equal(t, message.Attributes, received.Attributes)

		return nil
	}).Once()
	output := stream.NewSqsOutputWithInterfaces(logger, queue, &stream.SqsOutputSettings{})
	retry := stream.NewRetryHandlerSqsWithInterfaces(output, &stream.RetryHandlerSqsSettings{
		RetryHandlerSettings: stream.RetryHandlerSettings{After: time.Second},
	})
	require.NoError(t, retry.Put(t.Context(), message))

	for key, value := range before {
		require.Equal(t, value, message.Attributes[key])
	}
}

func TestMessageAggregateRoundTrip(t *testing.T) {
	original := []*stream.Message{{Body: "hello\x00世界", Attributes: map[string]string{"text": "valid UTF-8"}}}
	wire, err := json.Marshal(original)
	require.NoError(t, err)
	var restored []*stream.Message
	require.NoError(t, json.Unmarshal(wire, &restored))
	require.Equal(t, original, restored)
}

func TestMessageWireFormat(t *testing.T) {
	message := stream.Message{Body: "hello", Attributes: map[string]string{"foo": "bar"}}
	wire, err := message.MarshalToString()
	require.NoError(t, err)
	require.JSONEq(t, `{"attributes":{"foo":"bar"},"body":"hello"}`, wire)
	var received stream.Message
	require.NoError(t, received.UnmarshalFromString(wire))
	require.Equal(t, message, received)
	require.NoError(t, received.UnmarshalFromString(`{"attributes":{"foo":"bar"},"body":"hello"}`))
	require.Equal(t, message, received)

	for _, invalid := range []string{
		`{"attributes":{"number":42},"body":"hello"}`,
		`{"attributes":{"bool":true},"body":"hello"}`,
		`{"attributes":{},"body":42}`,
		`{"attributes":{},"body":true}`,
	} {
		require.Error(t, received.UnmarshalFromString(invalid))
	}
}

func TestMessageBodyRoundTrip(t *testing.T) {
	for _, attributes := range []map[string]string{nil, {stream.AttributeEncoding: stream.EncodingAvro.String()}} {
		message := stream.Message{Body: "hello\x00世界", Attributes: attributes}
		before := maps.Clone(attributes)

		for range 2 {
			wire, err := message.MarshalToBytes()
			require.NoError(t, err)
			var envelope stream.Message
			require.NoError(t, json.Unmarshal(wire, &envelope))
			require.Equal(t, message, envelope)
			require.Equal(t, before, message.Attributes)

			var restored stream.Message
			require.NoError(t, restored.UnmarshalFromBytes(wire))
			require.Equal(t, message, restored)
		}
	}
}

func TestMessagePreservesBase64Attribute(t *testing.T) {
	message := stream.Message{Body: "hello", Attributes: map[string]string{stream.AttributeBodyBase64: "true"}}
	wire, err := message.MarshalToBytes()
	require.NoError(t, err)
	require.JSONEq(t, `{"attributes":{"goso.body.base64":"true"},"body":"hello"}`, string(wire))
	var restored stream.Message
	require.NoError(t, restored.UnmarshalFromBytes(wire))
	require.Equal(t, message, restored)
}
