package stream

import (
	"github.com/justtrackio/gosoline/pkg/encoding/json"
)

const (
	AttributeSqsMessageId     = "sqsMessageId"
	AttributeSqsReceiptHandle = "sqsReceiptHandle"
)

type Message struct {
	Attributes map[string]interface{} `json:"attributes"`
	Body       string                 `json:"body"`
}

func (m *Message) GetAttributes() map[string]interface{} {
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
	return json.Unmarshal(data, m)
}

func (m *Message) UnmarshalFromString(data string) error {
	return m.UnmarshalFromBytes([]byte(data))
}
