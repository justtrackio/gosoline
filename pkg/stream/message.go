package stream

import (
	"encoding/base64"
	"fmt"
	"maps"
	"unicode/utf8"

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
	Body       string            `json:"body"`
}

// MarshalJSON preserves arbitrary payload bytes across JSON-based transports.
// Attributes are expected to contain valid UTF-8 strings.
func (m Message) MarshalJSON() ([]byte, error) {
	if _, ok := m.Attributes[AttributeBodyBase64]; ok {
		return nil, fmt.Errorf("message attribute %s is reserved for serialization", AttributeBodyBase64)
	}

	wire := wireMessage{
		Attributes: maps.Clone(m.Attributes),
		Body:       m.Body,
	}

	// if the body is not valid utf8, we have to base64 encode
	if !utf8.ValidString(m.Body) {
		wire.Body = base64.StdEncoding.EncodeToString([]byte(m.Body))
		if wire.Attributes == nil {
			wire.Attributes = make(map[string]string)
		}

		wire.Attributes[AttributeBodyBase64] = "true"
	}

	return json.Marshal(wire)
}

// UnmarshalJSON accepts UTF-8 and binary-safe envelopes,
// including messages nested inside aggregates.
func (m *Message) UnmarshalJSON(data []byte) error {
	return m.UnmarshalFromBytes(data)
}

func (m *Message) GetAttributes() map[string]string {
	return m.Attributes
}

func (m *Message) MarshalToBytes() ([]byte, error) {
	return json.Marshal(*m)
}

func (m *Message) MarshalToString() (string, error) {
	var err error
	var bytes []byte

	if bytes, err = m.MarshalToBytes(); err != nil {
		return "", err
	}

	return string(bytes), nil
}

func (m *Message) UnmarshalFromBytes(data []byte) error {
	var err error
	var body []byte

	var wire wireMessage
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}

	m.Attributes = wire.Attributes
	if m.Attributes == nil {
		m.Attributes = make(map[string]string)
	}

	m.Body = wire.Body

	if encoded, ok := m.Attributes[AttributeBodyBase64]; ok {
		if encoded != "true" {
			return fmt.Errorf("invalid base64 body flag %q", encoded)
		}

		if body, err = base64.StdEncoding.DecodeString(wire.Body); err != nil {
			return fmt.Errorf("can not decode base64 message body: %w", err)
		}

		m.Body = string(body)
		delete(m.Attributes, AttributeBodyBase64)
	}

	return nil
}

func (m *Message) UnmarshalFromString(data string) error {
	return m.UnmarshalFromBytes([]byte(data))
}

// wireMessage has the same fields as Message without its JSON methods.
type wireMessage Message
