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

func NewKafkaMessageAttrs(key string) map[string]any {
	return map[string]any{AttributeKafkaKey: key}
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

type kafkaMessageHandler struct {
	data chan *Message
}

func NewKafkaMessageHandler(data chan *Message) kafkaConsumer.KafkaMessageHandler {
	return &kafkaMessageHandler{
		data: data,
	}
}

func (h *kafkaMessageHandler) Handle(kafkaRecords []*kgo.Record) {
	for _, record := range kafkaRecords {
		if record == nil {
			continue
		}

		h.data <- KafkaToGosoMessage(*record)
	}
}

func (h *kafkaMessageHandler) Stop() {
	close(h.data)
}
