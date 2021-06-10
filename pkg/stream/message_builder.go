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
func GetEncodingAttribute(attributes map[string]interface{}) (*EncodingType, error) {
	if attrEncoding, ok := attributes[AttributeEncoding]; ok {
		// shortcut for unit tests which might specify the correct constant directly
		if encoding, ok := attrEncoding.(EncodingType); ok {
			return &encoding, nil
		}

		if encodingString, ok := attrEncoding.(string); ok {
			encoding := EncodingType(encodingString)

			return &encoding, nil
		}

		return nil, fmt.Errorf("the encoding attribute '%v' should be of type string but instead is '%T'", attrEncoding, attrEncoding)
	}

	return nil, nil
}

// GetCompressionAttribute returns the compression attribute if one is set, nil if none is set,
// and an error if the set value is of the wrong type.
func GetCompressionAttribute(attributes map[string]interface{}) (*CompressionType, error) {
	if attrCompression, ok := attributes[AttributeCompression]; ok {
		// shortcut for unit tests which might specify the correct constant directly
		if compression, ok := attrCompression.(CompressionType); ok {
			return &compression, nil
		}

		if compressionString, ok := attrCompression.(string); ok {
			compression := CompressionType(compressionString)

			return &compression, nil
		}

		return nil, fmt.Errorf("the compression attribute '%v' should be of type string but instead is '%T'", attrCompression, attrCompression)
	}

	return nil, nil
}

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
	data, err := NewJsonEncoder().Encode(body)

	if err != nil {
		return nil, fmt.Errorf("can not marshal body to json: %w", err)
	}

	msg := NewJsonMessage(string(data), attributes...)

	return msg, nil
}

func NewProtobufMessage(body string, attributes ...map[string]interface{}) *Message {
	msg := NewMessage(body, attributes...)
	msg.Attributes[AttributeEncoding] = EncodingProtobuf

	return msg
}

func MarshalProtobufMessage(body ProtobufEncodable, attributes ...map[string]interface{}) (*Message, error) {
	data, err := NewProtobufEncoder().Encode(body)

	if err != nil {
		return nil, fmt.Errorf("can not marshal body to protobuf: %w", err)
	}

	msg := NewProtobufMessage(string(data), attributes...)

	return msg, nil
}
