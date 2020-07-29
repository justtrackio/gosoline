package env

import (
	"fmt"
	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/encoding/json"
	"github.com/applike/gosoline/pkg/stream"
	"io/ioutil"
)

type streamInputComponent struct {
	baseComponent
	name  string
	input *stream.InMemoryInput
}

func (s *streamInputComponent) AppOptions() []application.Option {
	key := fmt.Sprintf("stream.input.%s.type", s.name)

	return []application.Option{
		application.WithConfigSetting(key, stream.InputTypeInMemory),
	}
}

func (s *streamInputComponent) Publish(body interface{}, attributes map[string]interface{}) {
	bytes, err := json.Marshal(body)

	if err != nil {
		s.failNow(err.Error(), "can not marshal message body for publishing")
	}

	message := &stream.Message{
		Attributes: attributes,
		Body:       string(bytes),
	}

	s.input.Publish(message)
}

func (s *streamInputComponent) PublishAndStop(body interface{}, attributes map[string]interface{}) {
	s.Publish(body, attributes)
	s.Stop()
}

func (s *streamInputComponent) PublishFromJsonFile(fileName string) {
	bytes, err := ioutil.ReadFile(fileName)

	if err != nil {
		s.failNow(err.Error(), "can not open json file to publish messages")
	}

	messages := make([]*stream.Message, 0)
	err = json.Unmarshal(bytes, &messages)

	if err != nil {
		s.failNow(err.Error(), "can not unmarshal messages from json file")
	}

	for _, msg := range messages {
		s.input.Publish(msg)
	}

	s.input.Stop()
}

func (s *streamInputComponent) Stop() {
	s.input.Stop()
}
