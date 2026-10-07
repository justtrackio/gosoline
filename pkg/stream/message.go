package stream

import (
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
