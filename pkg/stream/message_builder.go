package stream

import (
	"fmt"
)

const (
	AttributeEncoding    = "encoding"
	AttributeCompression = "compression"
)

// GetEncodingAttribute returns the encoding attribute if one is set, nil if none is set,
// and an error if the set value is of the wrong type.
func GetEncodingAttribute(attributes map[string]string) *EncodingType {
	if attrEncoding, ok := attributes[AttributeEncoding]; ok {
		encoding := EncodingType(attrEncoding)

		return &encoding
	}

	return nil
}

// GetCompressionAttribute returns the compression attribute if one is set, nil if none is set,
// and an error if the set value is of the wrong type.
func GetCompressionAttribute(attributes map[string]string) *CompressionType {
	if attrCompression, ok := attributes[AttributeCompression]; ok {
		compression := CompressionType(attrCompression)

		return &compression
	}

	return nil
}

func NewMessage(body string, attributes ...map[string]string) *Message {
	msg := &Message{
		Attributes: map[string]string{},
		Body:       body,
	}

	for _, attrs := range attributes {
		for k, v := range attrs {
			msg.Attributes[k] = v
		}
	}

	return msg
}

func NewJsonMessage(body string, attributes ...map[string]string) *Message {
	msg := NewMessage(body, attributes...)
	msg.Attributes[AttributeEncoding] = EncodingJson.String()

	return msg
}

func MarshalJsonMessage(body interface{}, attributes ...map[string]string) (*Message, error) {
	data, err := NewJsonEncoder().Encode(body)

	if err != nil {
		return nil, fmt.Errorf("can not marshal body to json: %w", err)
	}

	msg := NewJsonMessage(string(data), attributes...)

	return msg, nil
}

func NewProtobufMessage(body string, attributes ...map[string]string) *Message {
	msg := NewMessage(body, attributes...)
	msg.Attributes[AttributeEncoding] = EncodingProtobuf.String()

	return msg
}

func MarshalProtobufMessage(body ProtobufEncodable, attributes ...map[string]string) (*Message, error) {
	data, err := NewProtobufEncoder().Encode(body)

	if err != nil {
		return nil, fmt.Errorf("can not marshal body to protobuf: %w", err)
	}

	msg := NewProtobufMessage(string(data), attributes...)

	return msg, nil
}
