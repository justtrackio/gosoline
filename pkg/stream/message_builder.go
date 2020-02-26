package stream

import (
	"fmt"
	"github.com/applike/gosoline/pkg/encoding/json"
)

const (
	AttributeEncoding    = "encoding"
	AttributeCompression = "compression"
)

func NewMessage(body string, attributes ...map[string]interface{}) *Message {
	msg := &Message{
		Attributes: map[string]interface{}{},
		Body:       body,
	}

	for _, attrs := range attributes {
		for k, v := range attrs {
			msg.Attributes[k] = v
		}
	}

	return msg
}

func NewJsonMessage(body string, attributes ...map[string]interface{}) *Message {
	msg := NewMessage(body, attributes...)
	msg.Attributes[AttributeEncoding] = EncodingJson

	return msg
}

func MarshalJsonMessage(body interface{}, attributes ...map[string]interface{}) (*Message, error) {
	data, err := json.Marshal(body)

	if err != nil {
		return nil, fmt.Errorf("can not marshal body to json: %w", err)
	}

	msg := NewJsonMessage(string(data), attributes...)

	return msg, nil
}
