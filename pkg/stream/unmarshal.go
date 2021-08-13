package stream

import (
	"fmt"

	"github.com/applike/gosoline/pkg/cloud/aws/sns"
	"github.com/applike/gosoline/pkg/encoding/json"
)

const (
	UnmarshallerMsg = "msg"
	UnmarshallerRaw = "raw"
	UnmarshallerSns = "sns"
)

type UnmarshallerFunc func(data *string) (*Message, error)

var unmarshallers = map[string]UnmarshallerFunc{
	UnmarshallerMsg: MessageUnmarshaller,
	UnmarshallerRaw: RawUnmarshaller,
	UnmarshallerSns: SnsUnmarshaller,
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
