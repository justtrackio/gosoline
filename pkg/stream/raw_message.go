package stream

type Encoder interface {
	Encode(body interface{}) ([]byte, error)
}

type RawMessage struct {
	Body    interface{}
	Encoder Encoder
}

// Like NewRawMessage with the encoder set to marshal the body as JSON.
func NewRawJsonMessage(body interface{}) *RawMessage {
	return NewRawMessage(body, jsonEncoder{})
}

// Create a new RawMessage. It uses the provided encoder to encode the message body.
func NewRawMessage(body interface{}, encoder Encoder) *RawMessage {
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
