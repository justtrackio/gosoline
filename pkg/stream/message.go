package stream

import (
	"github.com/applike/gosoline/pkg/encoding/json"
)

const (
	AttributeSqsDelaySeconds   = "sqsDelaySeconds"
	AttributeSqsReceiptHandle  = "sqsReceiptHandle"
	AttributeSqsMessageGroupId = "sqsMessageGroupId"
)

type Message struct {
	Attributes map[string]interface{} `json:"attributes"`
	Body       string                 `json:"body"`
}

func (m *Message) MarshalToBytes() ([]byte, error) {
	return json.Marshal(*m)
}

func (m *Message) GetReceiptHandler() interface{} {
	var receiptHandleInterface interface{}
	var ok bool

	if receiptHandleInterface, ok = m.Attributes[AttributeSqsReceiptHandle]; !ok {
		return nil
	}

	return receiptHandleInterface
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
