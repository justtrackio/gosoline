package stream

import (
	"encoding/json"
	"fmt"
	"github.com/applike/gosoline/pkg/sns"
)

type MessageUnmarshaler func(data *string) (*Message, error)

func BasicUnmarshaler(data *string) (*Message, error) {
	msg := Message{}
	err := msg.UnmarshalFromString(*data)

	return &msg, err
}

func SnsUnmarshaler(data *string) (*Message, error) {
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
