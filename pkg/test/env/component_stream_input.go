package env

import (
	"fmt"
	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/encoding/json"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/stretchr/testify/assert"
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

func (s *streamInputComponent) PublishFromJsonFile(fileName string) {
	bytes, err := ioutil.ReadFile(fileName)

	if err != nil {
		assert.FailNow(s.t, "can not open json file to publish messages", err.Error())
	}

	messages := make([]*stream.Message, 0)
	err = json.Unmarshal(bytes, &messages)

	if err != nil {
		assert.FailNow(s.t, "can not unmarshal messages from json file", err.Error())
	}

	for _, msg := range messages {
		s.input.Publish(msg)
	}

	s.input.Stop()
}
