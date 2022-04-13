package stream

import (
	"encoding/json"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/protocol"
)

const (
	AttributeKafkaOriginalMessage = "KafkaOriginal"
	AttributeKafkaKey             = "KafkaKey"
)

type KafkaSourceMessage struct {
	kafka.Message
}

func (k KafkaSourceMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"Time":      k.Time,
		"Partition": k.Partition,
		"Offset":    k.Offset,
		"Headers":   KafkaToGosoAttributes(k.Headers, map[string]interface{}{}),
		"Key":       string(k.Key),
	})
}

func NewKafkaMessageAttrs(key string) map[string]interface{} {
	return map[string]interface{}{AttributeKafkaKey: key}
}

func KafkaToGosoAttributes(headers []kafka.Header, attributes map[string]interface{}) map[string]interface{} {
	for _, v := range headers {
		attributes[v.Key] = string(v.Value)
	}

	return attributes
}

func KafkaToGosoMessage(k kafka.Message) *Message {
	attributes := KafkaToGosoAttributes(k.Headers, map[string]interface{}{
		AttributeKafkaOriginalMessage: KafkaSourceMessage{Message: k},
	})

	return &Message{Body: string(k.Value), Attributes: attributes}
}

func GosoToKafkaMessages(msgs ...*Message) []kafka.Message {
	ks := []kafka.Message{}

	for _, m := range msgs {
		ks = append(ks, m.Attributes[AttributeKafkaOriginalMessage].(KafkaSourceMessage).Message)
	}

	return ks
}

func GosoToKafkaMessage(msg *Message) kafka.Message {
	return GosoToKafkaMessages(msg)[0]
}

func NewKafkaMessage(writable WritableMessage) kafka.Message {
	gMessage := writable.(*Message)
	kMessage := kafka.Message{Value: []byte(gMessage.Body)}

	key, ok := gMessage.GetAttributes()[AttributeKafkaKey].(string)
	if ok {
		kMessage.Key = []byte(key)
	}

	for k, v := range gMessage.Attributes {
		if k == AttributeKafkaKey {
			continue
		}
		vStr, ok := v.(string)
		if !ok {
			continue
		}

		kMessage.Headers = append(
			kMessage.Headers,
			protocol.Header{Key: k, Value: []byte(vStr)},
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
