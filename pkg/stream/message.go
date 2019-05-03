package stream

import (
	"context"
	"encoding/json"
	"github.com/applike/gosoline/pkg/tracing"
)

type Message struct {
	Trace      *tracing.Trace         `json:"trace"`
	Attributes map[string]interface{} `json:"attributes"`
	Body       string                 `json:"body"`
}

func (m *Message) GetTrace() *tracing.Trace {
	return m.Trace
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
	bytes := []byte(data)

	return json.Unmarshal(bytes, m)
}

func CreateMessage(ctx context.Context, body interface{}) (*Message, error) {
	msg := CreateMessageFromContext(ctx)

	serializedOutput, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	msg.Body = string(serializedOutput)

	return msg, nil
}

func CreateMessageFromContext(ctx context.Context) *Message {
	span := tracing.GetSpan(ctx)

	return &Message{
		Trace:      span.GetTrace(),
		Attributes: make(map[string]interface{}),
	}
}
