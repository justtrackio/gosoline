package stream

import (
	"fmt"

	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	AttributeKafkaKey = "KafkaKey"
)

func KafkaHeadersToGosoAttributes(kafkaRecordHeaders []kgo.RecordHeader) map[string]string {
	attributes := make(map[string]string)

	for _, v := range kafkaRecordHeaders {
		attributes[v.Key] = string(v.Value)
	}

	return attributes
}

func KafkaToGosoMessage(record kgo.Record) *Message {
	attributes := KafkaHeadersToGosoAttributes(record.Headers)

	if record.Key != nil {
		attributes[AttributeKafkaKey] = string(record.Key)
	}

	return &Message{Body: string(record.Value), Attributes: attributes}
}

func NewKafkaMessage(message WritableMessage) (*kgo.Record, error) {
	kafkaRecord := &kgo.Record{}
	var body []byte
	var attributes map[string]string

	// if the message comes from the producer daemon it's a rawJsonMessage that only holds the encoded model in the body
	// otherwise, it's a *Message
	switch m := message.(type) {
	case *Message:
		body = []byte(m.Body)
		attributes = m.Attributes
	case rawJsonMessage:
		body = m.body
		attributes = m.attributes
	default:
		return nil, fmt.Errorf("unexpected message type: %T", m)
	}

	kafkaRecord.Value = body

	key, ok := attributes[AttributeKafkaKey]
	if ok {
		kafkaRecord.Key = []byte(key)
	}

	for k, v := range attributes {
		if k == AttributeKafkaKey {
			continue
		}

		kafkaRecord.Headers = append(
			kafkaRecord.Headers,
			kgo.RecordHeader{Key: k, Value: []byte(v)},
		)
	}

	return kafkaRecord, nil
}

func NewKafkaMessages(messages []WritableMessage) ([]*kgo.Record, error) {
	var err error
	out := make([]*kgo.Record, len(messages))

	for i, message := range messages {
		if out[i], err = NewKafkaMessage(message); err != nil {
			return nil, fmt.Errorf("can not build kafka message: %w", err)
		}
	}

	return out, nil
}
