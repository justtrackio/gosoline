package stream

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/spf13/cast"
)

const (
	AttributeSqsMessageId               = "sqsMessageId"
	AttributeSqsReceiptHandle           = "sqsReceiptHandle"
	AttributeSqsApproximateReceiveCount = "sqsApproximateReceiveCount"
)

type Message struct {
	Attributes map[string]string      `json:"attributes"`
	Body       string                 `json:"body"`
	metaData   map[string]interface{} `json:"-"`
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
	type legacy struct {
		Attributes map[string]interface{} `json:"attributes"`
		Body       string                 `json:"body"`
	}

	legacyMsg := &legacy{}
	if err := json.Unmarshal(data, legacyMsg); err != nil {
		return err
	}

	m.Attributes = make(map[string]string)
	m.Body = legacyMsg.Body

	var err error
	for k, v := range legacyMsg.Attributes {
		if m.Attributes[k], err = cast.ToStringE(v); err != nil {
			return fmt.Errorf("can not cast attribute %s=%v to string: %w", k, v, err)
		}
	}

	return nil
}

func (m *Message) UnmarshalFromString(data string) error {
	return m.UnmarshalFromBytes([]byte(data))
}
