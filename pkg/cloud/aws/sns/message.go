package sns

import "github.com/justtrackio/gosoline/pkg/encoding/json"

type Message struct {
	Type      string
	TopicArn  string
	MessageId string
	Message   string
}

func (m Message) MarshalToBytes() ([]byte, error) {
	return json.Marshal(m)
}

func (m Message) MarshalToString() (string, error) {
	bytes, err := json.Marshal(m)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}
