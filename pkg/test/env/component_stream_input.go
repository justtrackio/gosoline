package env

import (
	"fmt"
	"io/ioutil"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/justtrackio/gosoline/pkg/stream"
)

type StreamInputComponent struct {
	baseComponent
	name  string
	input *stream.InMemoryInput
}

func (s *StreamInputComponent) CfgOptions() []cfg.Option {
	key := fmt.Sprintf("stream.input.%s.type", s.name)

	return []cfg.Option{
		cfg.WithConfigSetting(key, stream.InputTypeInMemory),
	}
}

func (s *StreamInputComponent) Publish(body interface{}, attributes map[string]interface{}) {
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

func (s *StreamInputComponent) PublishAndStop(body interface{}, attributes map[string]interface{}) {
	s.Publish(body, attributes)
	s.Stop()
}

func (s *StreamInputComponent) PublishFromJsonFile(fileName string) {
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

func (s *StreamInputComponent) Stop() {
	s.input.Stop()
}
