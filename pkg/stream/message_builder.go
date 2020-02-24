package stream

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
