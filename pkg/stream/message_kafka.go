package stream

import (
	"encoding/json"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/protocol"
)

const (
	AttributeKafkaKey            = "KafkaKey"
	MetaDataKafkaOriginalMessage = "KafkaOriginal"
)

type KafkaSourceMessage struct {
	kafka.Message
}

func (k KafkaSourceMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"Time":      k.Time,
		"Partition": k.Partition,
		"Offset":    k.Offset,
		"Headers":   KafkaHeadersToGosoAttributes(k.Headers),
		"Key":       string(k.Key),
	})
}

func NewKafkaMessageAttrs(key string) map[string]interface{} {
	return map[string]interface{}{AttributeKafkaKey: key}
}

func KafkaHeadersToGosoAttributes(headers []kafka.Header) map[string]string {
	attributes := make(map[string]string)

	for _, v := range headers {
		attributes[v.Key] = string(v.Value)
	}

	return attributes
}

func KafkaToGosoMessage(k kafka.Message) *Message {
	attributes := KafkaHeadersToGosoAttributes(k.Headers)
	metaData := map[string]interface{}{
		MetaDataKafkaOriginalMessage: KafkaSourceMessage{Message: k},
	}

	return &Message{Body: string(k.Value), Attributes: attributes, metaData: metaData}
}

func GosoToKafkaMessages(msgs ...*Message) []kafka.Message {
	ks := []kafka.Message{}

	for _, m := range msgs {
		ks = append(ks, GosoToKafkaMessage(m))
	}

	return ks
}

func GosoToKafkaMessage(msg *Message) kafka.Message {
	return msg.metaData[MetaDataKafkaOriginalMessage].(KafkaSourceMessage).Message
}

func NewKafkaMessage(writable WritableMessage) kafka.Message {
	gMessage := writable.(*Message)
	kMessage := kafka.Message{Value: []byte(gMessage.Body)}

	key, ok := gMessage.GetAttributes()[AttributeKafkaKey]
	if ok {
		kMessage.Key = []byte(key)
	}

	for k, v := range gMessage.Attributes {
		if k == AttributeKafkaKey {
			continue
		}

		kMessage.Headers = append(
			kMessage.Headers,
			protocol.Header{Key: k, Value: []byte(v)},
		)
	}

	return kMessage
}

func NewKafkaMessages(ms []WritableMessage) []kafka.Message {
	out := []kafka.Message{}
	for _, m := range ms {
		out = append(out, NewKafkaMessage(m))
	}

	return out
}
