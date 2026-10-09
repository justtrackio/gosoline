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

func TestBinaryKafkaMessageSurvivesSqsRetry(t *testing.T) {
	record := kgo.Record{
		Value: []byte{0, 0, 0, 0, 0x80, 0xff, 1},
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

func TestMessageBinaryAggregateRoundTrip(t *testing.T) {
	original := []*stream.Message{{Body: string([]byte{0xff, 0}), Attributes: map[string]string{"text": "valid UTF-8"}}}
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

	for _, invalid := range []string{
		`{"attributes":{"number":42},"body":"hello"}`,
		`{"attributes":{"bool":true},"body":"hello"}`,
		`{"attributes":{"goso.body.base64":"true"},"body":"!"}`,
		`{"attributes":{"goso.body.base64":"false"},"body":"hello"}`,
	} {
		require.Error(t, received.UnmarshalFromString(invalid))
	}
}

func TestMessageBase64BodyAttribute(t *testing.T) {
	for _, attributes := range []map[string]string{nil, {stream.AttributeEncoding: stream.EncodingAvro.String()}} {
		message := stream.Message{Body: string([]byte{0xff, 0}), Attributes: attributes}
		before := maps.Clone(attributes)

		for range 2 {
			wire, err := message.MarshalToBytes()
			require.NoError(t, err)
			var envelope struct {
				Attributes map[string]string `json:"attributes"`
				Body       string            `json:"body"`
			}
			require.NoError(t, json.Unmarshal(wire, &envelope))
			require.Equal(t, "/wA=", envelope.Body)
			require.Equal(t, "true", envelope.Attributes[stream.AttributeBodyBase64])
			require.NotContains(t, string(wire), "bodyBase64")
			require.Equal(t, before, message.Attributes)

			var restored stream.Message
			require.NoError(t, restored.UnmarshalFromBytes(wire))
			require.Equal(t, message.Body, restored.Body)
			require.NotContains(t, restored.Attributes, stream.AttributeBodyBase64)

			for key, value := range attributes {
				require.Equal(t, value, restored.Attributes[key])
			}
		}
	}
}

func TestMessageRejectsReservedBase64Attribute(t *testing.T) {
	message := stream.Message{Body: "hello", Attributes: map[string]string{stream.AttributeBodyBase64: "true"}}
	_, err := message.MarshalToBytes()
	require.ErrorContains(t, err, "reserved for serialization")
}
