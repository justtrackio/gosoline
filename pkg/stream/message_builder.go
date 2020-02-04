package stream

const AttributeEncoding = "encoding"

func NewMessage(body string) *Message {
	return &Message{
		Attributes: map[string]interface{}{},
		Body:       body,
	}
}

func NewMessageWithAttributes(body string, attributes map[string]interface{}) *Message {
	return &Message{
		Attributes: attributes,
		Body:       body,
	}
}

func NewJsonMessage(body string) *Message {
	return &Message{
		Attributes: map[string]interface{}{
			AttributeEncoding: EncodingJson,
		},
		Body: body,
	}
}
