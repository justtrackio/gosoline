package stream

import (
	"encoding/base64"
	"fmt"
	"maps"

	"github.com/justtrackio/gosoline/pkg/encoding/json"
)

const (
	// AttributeBodyBase64 marks a JSON envelope's body as base64-encoded. It is
	// reserved for serialization and removed when the message is unmarshalled.
	AttributeBodyBase64                 = "goso.body.base64"
	AttributeSqsMessageId               = "sqsMessageId"
	AttributeSqsReceiptHandle           = "sqsReceiptHandle"
	AttributeSqsApproximateReceiveCount = "sqsApproximateReceiveCount"
)

type Message struct {
	Attributes map[string]string `json:"attributes"`
	Body       []byte            `json:"body"`
}

// MarshalJSON writes a base64 body and a temporary migration marker. The marker
// belongs to the JSON envelope only; the message's attributes are not modified.
func (m Message) MarshalJSON() ([]byte, error) {
	attributes := maps.Clone(m.Attributes)
	if attributes == nil {
		attributes = make(map[string]string)
	}

	attributes[AttributeBodyBase64] = "true"

	// The distinct type has no MarshalJSON method, preventing recursive calls.
	type envelope Message

	return json.Marshal(envelope{Attributes: attributes, Body: m.Body})
}

// UnmarshalJSON accepts marked base64 bodies and legacy plain string bodies.
func (m *Message) UnmarshalJSON(data []byte) error {
	type jsonEnvelope struct {
		Attributes map[string]string `json:"attributes"`
		Body       json.RawMessage   `json:"body"`
	}

	// Decode a copy so malformed envelopes cannot mutate the original attributes.
	envelope := jsonEnvelope{Attributes: maps.Clone(m.Attributes)}

	if err := json.Unmarshal(data, &envelope); err != nil {
		return err
	}

	marker, marked := envelope.Attributes[AttributeBodyBase64]
	if marked && marker != "true" {
		return fmt.Errorf("invalid %s marker: %q", AttributeBodyBase64, marker)
	}

	body := m.Body
	if len(envelope.Body) > 0 {
		var err error
		if body, err = m.unmarshalBody(envelope.Body, marked); err != nil {
			return err
		}
	}

	delete(envelope.Attributes, AttributeBodyBase64)
	m.Attributes = envelope.Attributes
	m.Body = body

	return nil
}

func (m *Message) unmarshalBody(data json.RawMessage, marked bool) ([]byte, error) {
	var encodedBody *string
	if err := json.Unmarshal(data, &encodedBody); err != nil {
		return nil, err
	}

	if encodedBody == nil {
		return nil, nil
	}

	if !marked {
		return []byte(*encodedBody), nil
	}

	var body []byte
	var err error

	if body, err = base64.StdEncoding.DecodeString(*encodedBody); err != nil {
		return nil, fmt.Errorf("could not decode base64 message body: %w", err)
	}

	return body, nil
}

func (m *Message) GetAttributes() map[string]string {
	return m.Attributes
}

func (m *Message) MarshalToBytes() ([]byte, error) {
	return json.Marshal(*m)
}

func (m *Message) MarshalToString() (string, error) {
	var bytes []byte
	var err error

	if bytes, err = json.Marshal(*m); err != nil {
		return "", err
	}

	return string(bytes), nil
}

func (m *Message) UnmarshalFromBytes(data []byte) error {
	if err := json.Unmarshal(data, m); err != nil {
		return err
	}

	return nil
}

func (m *Message) UnmarshalFromString(data string) error {
	return m.UnmarshalFromBytes([]byte(data))
}
