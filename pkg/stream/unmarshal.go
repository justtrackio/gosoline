package stream

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cloud/aws/sns"
	"github.com/justtrackio/gosoline/pkg/encoding/base64"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
)

const (
	UnmarshallerMsg       = "msg"
	UnmarshallerRaw       = "raw"
	UnmarshallerSns       = "sns"
	UnmarshallerMsgBase64 = "msg_base64"
)

type UnmarshallerFunc func(data *string) (*Message, error)

var unmarshallers = map[string]UnmarshallerFunc{
	UnmarshallerMsg:       MessageUnmarshaller,
	UnmarshallerRaw:       RawUnmarshaller,
	UnmarshallerSns:       SnsUnmarshaller,
	UnmarshallerMsgBase64: MessageBase64Unmarshaller,
}

func MessageUnmarshaller(data *string) (*Message, error) {
	msg := Message{}
	err := msg.UnmarshalFromString(*data)

	return &msg, err
}

func RawUnmarshaller(data *string) (*Message, error) {
	return &Message{
		Body: *data,
	}, nil
}

func SnsMarshaller(msg *Message) (*string, error) {
	bytes, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	snsMessage := sns.Message{
		Type:    "Notification",
		Message: string(bytes),
	}

	bytes, err = json.Marshal(snsMessage)
	if err != nil {
		return nil, err
	}

	data := string(bytes)

	return &data, nil
}

func SnsUnmarshaller(data *string) (*Message, error) {
	bytes := []byte(*data)

	snsMessage := sns.Message{}
	err := json.Unmarshal(bytes, &snsMessage)
	if err != nil {
		return nil, err
	}

	if snsMessage.Type != "Notification" {
		return nil, fmt.Errorf("the sns message should be of the type 'Notification'")
	}

	msg := Message{}
	err = msg.UnmarshalFromString(snsMessage.Message)

	return &msg, err
}

func MessageBase64Unmarshaller(data *string) (*Message, error) {
	msg := &Message{}

	err := json.Unmarshal([]byte(*data), msg)
	if err != nil {
		return nil, err
	}

	bytes, err := base64.DecodeString(msg.Body)
	if err != nil {
		return nil, err
	}

	msg.Body = string(bytes)

	return msg, nil
}
