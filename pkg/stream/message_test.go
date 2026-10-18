package stream_test

import (
	"context"
	"encoding/base64"
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

	// A pre-transition reader sees the same body and attributes on the wire.
	type legacyMessage struct {
		Attributes map[string]string `json:"attributes"`
		Body       string            `json:"body"`
	}

	var legacy legacyMessage
	require.NoError(t, json.Unmarshal([]byte(wire), &legacy))
	require.Equal(t, message.Body, legacy.Body)
	require.Equal(t, message.Attributes, legacy.Attributes)
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
		`{"attributes":{"goso.body.base64":"true"},"body":"not base64!"}`,
		`{"attributes":{"goso.body.base64":"false"},"body":"hello"}`,
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

func TestMessageReservesBase64Attribute(t *testing.T) {
	var restored stream.Message
	require.NoError(t, restored.UnmarshalFromString(`{"attributes":{"goso.body.base64":"true","encoding":"application/json"},"body":"aGVsbG8="}`))
	require.Equal(t, "hello", restored.Body)
	require.NotContains(t, restored.Attributes, stream.AttributeBodyBase64)

	// Forwarding/retrying a decoded message still writes the legacy format.
	wire, err := restored.MarshalToString()
	require.NoError(t, err)
	require.JSONEq(t, `{"attributes":{"encoding":"application/json"},"body":"hello"}`, wire)
}

func TestMessageLegacyBodies(t *testing.T) {
	for _, body := range []string{`{"foo":"bar"}`, `[1,2]`, `"text"`, "42", "true", "null", "aGVsbG8=", "H4sIAAAAAAAA/", ""} {
		t.Run(body, func(t *testing.T) {
			wire, err := json.Marshal(map[string]any{"attributes": map[string]string{"encoding": "application/json"}, "body": body})
			require.NoError(t, err)
			var message stream.Message
			require.NoError(t, message.UnmarshalFromBytes(wire))
			require.Equal(t, body, message.Body)
			require.Equal(t, "application/json", message.Attributes[stream.AttributeEncoding])
		})
	}
}

func TestMessageReadsBase64Bodies(t *testing.T) {
	for _, body := range [][]byte{nil, {}, []byte("hello\x00世界"), {0, 0xff, 0xfe}} {
		wire, err := json.Marshal(map[string]any{
			"attributes": map[string]string{stream.AttributeBodyBase64: "true", "encoding": "application/avro"},
			"body":       body,
		})
		require.NoError(t, err)
		var received stream.Message
		require.NoError(t, received.UnmarshalFromBytes(wire))
		require.Equal(t, string(body), received.Body)
		require.Equal(t, map[string]string{"encoding": "application/avro"}, received.Attributes)
	}
}

func TestMessageMixedAggregateFormats(t *testing.T) {
	var messages []*stream.Message
	require.NoError(t, json.Unmarshal([]byte(`[{"attributes":{},"body":"aGVsbG8="},{"attributes":{"goso.body.base64":"true"},"body":"AP8="}]`), &messages))
	require.Equal(t, "aGVsbG8=", messages[0].Body)
	require.Equal(t, string([]byte{0, 0xff}), messages[1].Body)
	require.NotContains(t, messages[1].Attributes, stream.AttributeBodyBase64)
}

func TestMessageEncodingSurvivesEnvelopeMigration(t *testing.T) {
	for _, compression := range []stream.CompressionType{stream.CompressionNone, stream.CompressionGZip} {
		t.Run(compression.String(), func(t *testing.T) {
			encoder := stream.NewMessageEncoder(&stream.MessageEncoderSettings{
				Encoding:    stream.EncodingJson,
				Compression: compression,
			})
			model := map[string]string{"foo": "bar"}
			message, err := encoder.Encode(t.Context(), model)
			require.NoError(t, err)

			legacy, err := message.MarshalToBytes()
			require.NoError(t, err)
			attributes := maps.Clone(message.Attributes)
			attributes[stream.AttributeBodyBase64] = "true"
			marked, err := json.Marshal(map[string]any{"attributes": attributes, "body": base64.StdEncoding.EncodeToString([]byte(message.Body))})
			require.NoError(t, err)

			for _, wire := range [][]byte{legacy, marked} {
				var received stream.Message
				require.NoError(t, received.UnmarshalFromBytes(wire))
				require.Equal(t, message, &received)
				var restored map[string]string
				_, _, err = encoder.Decode(t.Context(), &received, &restored)
				require.NoError(t, err)
				require.Equal(t, model, restored)
			}
		})
	}
}

func TestMessageUnmarshalPreservesExistingFields(t *testing.T) {
	message := stream.Message{Body: "hello", Attributes: map[string]string{"foo": "bar"}}
	for _, wire := range []string{`{}`, `null`, `{"body":null}`} {
		require.NoError(t, message.UnmarshalFromString(wire))
		require.Equal(t, "hello", message.Body)
		require.Equal(t, map[string]string{"foo": "bar"}, message.Attributes)
	}

	require.Error(t, message.UnmarshalFromString(`{"attributes":{"goso.body.base64":"true"},"body":"invalid!"}`))
	require.Equal(t, "hello", message.Body)
	require.Equal(t, map[string]string{"foo": "bar"}, message.Attributes)
}
