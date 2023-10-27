package stream_test

import (
	"encoding/json"
	"testing"

	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/stretchr/testify/assert"

	"github.com/segmentio/kafka-go"
)

func Test_NewKafkaMessageAttrs(t *testing.T) {
	assert.Equal(t, stream.NewKafkaMessageAttrs("MyKey"), map[string]interface{}{"KafkaKey": "MyKey"})
}

func Test_MessageConversion(t *testing.T) {
	var (
		kMessage1 = kafka.Message{
			Key:   []byte("MessageKey1"),
			Value: []byte("MessageValue1"),
			Headers: []kafka.Header{
				{
					Key:   "HeaderKey1",
					Value: []byte("HeaderValue1"),
				},
			},
		}

		kMessage2 = kafka.Message{
			Key:   []byte("MessageKey2"),
			Value: []byte("MessageValue2"),
			Headers: []kafka.Header{
				{
					Key:   "HeaderKey2",
					Value: []byte("HeaderValue2"),
				},
			},
		}
	)

	gMessage1 := stream.KafkaToGosoMessage(kMessage1)
	gMessage2 := stream.KafkaToGosoMessage(kMessage2)

	assert.Equal(t,
		[]kafka.Message{
			kMessage1,
			kMessage2,
		},
		stream.GosoToKafkaMessages(gMessage1, gMessage2),
	)
}

func Test_GosoMessageSerialization(t *testing.T) {
	kMessage := kafka.Message{
		Key:   []byte("MessageKey"),
		Value: []byte("MessageValue"),
		Headers: []kafka.Header{
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
	gMessage := &stream.Message{
		Body: `{"MessageContent": "Content"}`,
		Attributes: map[string]string{
			"Attr1":    "1",
			"Attr2":    "2",
			"KafkaKey": "MyKey",
		},
	}

	var (
		expected = kafka.Message{
			Key:   []byte("MyKey"),
			Value: []byte(`{"MessageContent": "Content"}`),
			Headers: []kafka.Header{
				{
					Key:   "Attr1",
					Value: []byte("1"),
				},
				{
					Key:   "Attr2",
					Value: []byte("2"),
				},
			}}
		actual = stream.NewKafkaMessage(gMessage)
	)

	assert.Equal(
		t,
		expected.Key,
		actual.Key,
	)

	assert.Equal(
		t,
		expected.Value,
		actual.Value,
	)

	assert.ElementsMatch(
		t,
		expected.Headers,
		actual.Headers,
	)
}

func Test_NewKafkaMessages(t *testing.T) {
	var (
		gMessage1 = &stream.Message{
			Body: `{"MessageContent": "Content1"}`,
			Attributes: map[string]string{
				"Attr11": "11",
			},
		}
		gMessage2 = &stream.Message{
			Body: `{"MessageContent": "Content2"}`,
			Attributes: map[string]string{
				"Attr11": "12",
			},
		}
	)

	assert.Equal(t, []kafka.Message{
		stream.NewKafkaMessage(gMessage1),
		stream.NewKafkaMessage(gMessage2),
	},
		stream.NewKafkaMessages([]stream.WritableMessage{gMessage1, gMessage2}),
	)
}
