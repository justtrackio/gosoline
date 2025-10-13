package stream_test

import (
	"encoding/json"
	"testing"

	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/stretchr/testify/assert"
	"github.com/twmb/franz-go/pkg/kgo"
)

func Test_NewKafkaMessageAttrs(t *testing.T) {
	assert.Equal(t, stream.NewKafkaMessageAttrs("MyKey"), map[string]any{"KafkaKey": "MyKey"})
}

func Test_GosoMessageSerialization(t *testing.T) {
	kMessage := kgo.Record{
		Key:   []byte("MessageKey"),
		Value: []byte("MessageValue"),
		Headers: []kgo.RecordHeader{
			{
				Key:   "HeaderKey",
				Value: []byte("HeaderValue"),
			},
		},
	}
	gMessage := stream.KafkaToGosoMessage(kMessage)

	serialized, err := json.Marshal(gMessage)
	assert.Nil(t, err)

	assert.JSONEq(t, string(serialized), `{
		"attributes": {"HeaderKey":"HeaderValue"},
		"body": "MessageValue"
	}`)
}

func Test_NewKafkaMessage(t *testing.T) {
	attributes := map[string]string{
		"Attr1":    "1",
		"Attr2":    "2",
		"KafkaKey": "MyKey",
	}
	body := `{"MessageContent": "Content"}`

	gMessage := &stream.Message{
		Body:       body,
		Attributes: attributes,
	}

	gRawJsonMessage := stream.NewRawJsonMessage(attributes, []byte(body))

	expected := kgo.Record{
		Key:   []byte("MyKey"),
		Value: []byte(`{"MessageContent": "Content"}`),
		Headers: []kgo.RecordHeader{
			{
				Key:   "Attr1",
				Value: []byte("1"),
			},
			{
				Key:   "Attr2",
				Value: []byte("2"),
			},
		},
	}

	actualFromMessage, err := stream.NewKafkaMessage(gMessage)
	assert.NoError(t, err)
	assert.Equal(t, expected.Key, actualFromMessage.Key)
	assert.Equal(t, expected.Value, actualFromMessage.Value)
	assert.ElementsMatch(t, expected.Headers, actualFromMessage.Headers)

	actualFromRawJsonMessage, err := stream.NewKafkaMessage(gRawJsonMessage)
	assert.NoError(t, err)
	assert.Equal(t, expected.Key, actualFromRawJsonMessage.Key)
	assert.Equal(t, expected.Value, actualFromRawJsonMessage.Value)
	assert.ElementsMatch(t, expected.Headers, actualFromRawJsonMessage.Headers)
}

func Test_NewKafkaMessages(t *testing.T) {
	gMessage1 := &stream.Message{
		Body: `{"MessageContent": "Content1"}`,
		Attributes: map[string]string{
			"Attr11": "11",
		},
	}
	gMessage2 := &stream.Message{
		Body: `{"MessageContent": "Content2"}`,
		Attributes: map[string]string{
			"Attr11": "12",
		},
	}

	record1, err := stream.NewKafkaMessage(gMessage1)
	assert.NoError(t, err)

	record2, err := stream.NewKafkaMessage(gMessage2)
	assert.NoError(t, err)

	records, err := stream.NewKafkaMessages([]stream.WritableMessage{gMessage1, gMessage2})
	assert.NoError(t, err)

	assert.Equal(t, []*kgo.Record{
		record1,
		record2,
	},
		records,
	)
}
