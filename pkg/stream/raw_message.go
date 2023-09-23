package stream

type RawMessage struct {
	Body    interface{}
	Encoder MessageBodyEncoder
}

// NewRawJsonMessage works like NewRawMessage with the encoder set to marshal the body as JSON.
func NewRawJsonMessage(body interface{}) *RawMessage {
	return NewRawMessage(body, jsonEncoder{})
}

// NewRawMessage creates a new RawMessage. It uses the provided encoder to encode the message body.
func NewRawMessage(body interface{}, encoder MessageBodyEncoder) *RawMessage {
	return &RawMessage{
		Body:    body,
		Encoder: encoder,
	}
}

func (m *RawMessage) MarshalToBytes() ([]byte, error) {
	return m.Encoder.Encode(m.Body)
}

func (m *RawMessage) MarshalToString() (string, error) {
	bytes, err := m.MarshalToBytes()
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}
