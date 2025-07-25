package stream

import (
	"fmt"

	kafkaConsumer "github.com/justtrackio/gosoline/pkg/kafka/consumer"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	AttributeKafkaKey            = "KafkaKey"
	MetaDataKafkaOriginalMessage = "KafkaOriginal"
)

type KafkaSourceMessage struct {
	kgo.Record
}

func KafkaHeadersToGosoAttributes(kafkaRecordHeaders []kgo.RecordHeader) map[string]string {
	attributes := make(map[string]string)

	for _, v := range kafkaRecordHeaders {
		attributes[v.Key] = string(v.Value)
	}

	return attributes
}

func KafkaToGosoMessage(kafkaRecord kgo.Record) *Message {
	attributes := KafkaHeadersToGosoAttributes(kafkaRecord.Headers)
	metaData := map[string]any{
		MetaDataKafkaOriginalMessage: KafkaSourceMessage{Record: kafkaRecord},
	}

	return &Message{Body: string(kafkaRecord.Value), Attributes: attributes, metaData: metaData}
}

func NewKafkaMessage(message WritableMessage) (*kgo.Record, error) {
	kafkaRecord := &kgo.Record{}
	var body []byte

	// if the message comes from the producer daemon it's a rawJsonMessage
	// otherwise, it's a *Message
	switch m := message.(type) {
	case *Message:
		body = []byte(m.Body)
	case rawJsonMessage:
		// the kafka output does not support aggregation, so we can expect a single message in here
		msg := Message{}
		if err := msg.UnmarshalFromBytes(m.body); err != nil {
			return nil, fmt.Errorf("failed to unmarshal message body: %w", err)
		}
		body = []byte(msg.Body)
	default:
		return nil, fmt.Errorf("unexpected message type: %T", m)
	}

	kafkaRecord.Value = body
	attributes := getAttributes(message)

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

type kafkaMessageHandler struct {
	data chan *Message
}

func NewKafkaMessageHandler(data chan *Message) kafkaConsumer.KafkaMessageHandler {
	return &kafkaMessageHandler{
		data: data,
	}
}

func (s kafkaMessageHandler) Handle(kafkaRecords []*kgo.Record) {
	for _, record := range kafkaRecords {
		if record == nil {
			continue
		}

		s.data <- KafkaToGosoMessage(*record)
	}
}
