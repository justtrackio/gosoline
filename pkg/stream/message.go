package stream

import (
	"github.com/applike/gosoline/pkg/encoding/json"
)

const (
	AttributeSqsMessageId     = "sqsMessageId"
	AttributeSqsReceiptHandle = "sqsReceiptHandle"
)

type Message struct {
	Attributes map[string]interface{} `json:"attributes"`
	Body       string                 `json:"body"`
}

func (m *Message) WithGzipCompression() *Message {
	m.Attributes[AttributeCompression] = messageBodyCompressors[CompressionGZip]

	return m
}

func (m *Message) MarshalToBytes() ([]byte, error) {
	return json.Marshal(*m)
}

func (m *Message) GetReceiptHandler() interface{} {
	if receiptHandle, ok := m.Attributes[AttributeSqsReceiptHandle]; ok {
		return receiptHandle
	}

	return nil
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
