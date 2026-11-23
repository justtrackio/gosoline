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
	Body       string            `json:"body"`
}

// UnmarshalJSON accepts marked base64 bodies and legacy plain string bodies.
// Writers retain the legacy format during this reader-only transition release.
func (m *Message) UnmarshalJSON(data []byte) error {
	type jsonEnvelope Message

	// Seed the envelope to retain the usual JSON behavior for absent/null fields
	// when unmarshalling into an existing message. Decode attributes separately so
	// stripping the migration marker cannot mutate the original map on an error.
	envelope := jsonEnvelope(*m)
	envelope.Attributes = maps.Clone(m.Attributes)

	if err := json.Unmarshal(data, &envelope); err != nil {
		return err
	}

	marker, marked := envelope.Attributes[AttributeBodyBase64]
	if marked && marker != "true" {
		return fmt.Errorf("invalid %s marker: %q", AttributeBodyBase64, marker)
	}

	if marked {
		var body []byte
		var err error

		if body, err = base64.StdEncoding.DecodeString(envelope.Body); err != nil {
			return fmt.Errorf("could not decode base64 message body: %w", err)
		}

		envelope.Body = string(body)
	}

	delete(envelope.Attributes, AttributeBodyBase64)
	m.Attributes = envelope.Attributes
	m.Body = envelope.Body

	return nil
}

func (m *Message) GetAttributes() map[string]string {
	return m.Attributes
}

func (m *Message) MarshalToBytes() ([]byte, error) {
	return json.Marshal(*m)
}

func (m *Message) MarshalToString() (string, error) {
	bytes, err := json.Marshal(*m)
	if err != nil {
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
