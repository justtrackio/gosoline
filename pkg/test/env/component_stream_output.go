package env

import (
	"fmt"
	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/encoding/json"
	"github.com/applike/gosoline/pkg/stream"
)

type streamOutputComponent struct {
	baseComponent
	name   string
	output *stream.InMemoryOutput
}

func (s *streamOutputComponent) AppOptions() []application.Option {
	key := fmt.Sprintf("stream.output.%s.type", s.name)

	return []application.Option{
		application.WithConfigSetting(key, stream.OutputTypeInMemory),
	}
}

func (s *streamOutputComponent) Len() int {
	return s.output.Len()
}

func (s *streamOutputComponent) Get(i int) (*stream.Message, bool) {
	return s.output.Get(i)
}

func (s *streamOutputComponent) Unmarshal(i int, output interface{}) map[string]interface{} {
	msg, ok := s.Get(i)

	if !ok {
		s.failNow("message not available", "there is no message with index %d", i)
	}

	if err := json.Unmarshal([]byte(msg.Body), output); err != nil {
		s.failNow(err.Error(), "can not unmarshal message body")
	}

	return msg.Attributes
}
