package stream

import (
	"encoding/base64"
	"fmt"
	"maps"
	"unicode/utf8"

	"github.com/justtrackio/gosoline/pkg/encoding/json"
)

const (
	AttributeSqsMessageId               = "sqsMessageId"
	AttributeSqsReceiptHandle           = "sqsReceiptHandle"
	AttributeSqsApproximateReceiveCount = "sqsApproximateReceiveCount"
)

type Message struct {
	Attributes map[string]string `json:"attributes"`
	Body       string            `json:"body"`
}

// MarshalJSON preserves arbitrary payload and attribute bytes across JSON-based
// transports. UTF-8 messages retain their existing wire representation.
func (m Message) MarshalJSON() ([]byte, error) {
	wire := wireMessage{
		Attributes: m.Attributes,
		Body:       m.Body,
	}

	// if the body is not valid utf8, we have to base64 encode
	if !utf8.ValidString(m.Body) {
		wire.Body = ""
		wire.BodyBase64 = base64.StdEncoding.EncodeToString([]byte(m.Body))
	}

	for key, value := range m.Attributes {
		if utf8.ValidString(value) {
			continue
		}

		if wire.AttributesBase64 == nil {
			wire.AttributesBase64 = make(map[string]string)
			wire.Attributes = make(map[string]string, len(m.Attributes))
			maps.Copy(wire.Attributes, m.Attributes)
		}

		delete(wire.Attributes, key)
		wire.AttributesBase64[key] = base64.StdEncoding.EncodeToString([]byte(value))
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
	var decoded []byte

	var wire wireMessage
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}

	m.Attributes = wire.Attributes
	if m.Attributes == nil {
		m.Attributes = make(map[string]string)
	}
	m.Body = wire.Body

	if wire.BodyBase64 != "" {
		if body, err = base64.StdEncoding.DecodeString(wire.BodyBase64); err != nil {
			return fmt.Errorf("can not decode base64 message body: %w", err)
		}

		m.Body = string(body)
	}

	for key, value := range wire.AttributesBase64 {
		if decoded, err = base64.StdEncoding.DecodeString(value); err != nil {
			return fmt.Errorf("can not decode base64 attribute %s: %w", key, err)
		}

		m.Attributes[key] = string(decoded)
	}

	return nil
}

func (m *Message) UnmarshalFromString(data string) error {
	return m.UnmarshalFromBytes([]byte(data))
}

type wireMessage struct {
	Attributes       map[string]string `json:"attributes"`
	Body             string            `json:"body"`
	BodyBase64       string            `json:"bodyBase64,omitempty"`
	AttributesBase64 map[string]string `json:"attributesBase64,omitempty"`
}
