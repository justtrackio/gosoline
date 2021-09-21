package env

import (
	"context"
	"fmt"
	"reflect"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/stream"
)

type streamOutputComponent struct {
	baseComponent
	name    string
	output  *stream.InMemoryOutput
	encoder stream.MessageEncoder
}

func (s *streamOutputComponent) CfgOptions() []cfg.Option {
	key := fmt.Sprintf("stream.output.%s.type", s.name)

	return []cfg.Option{
		cfg.WithConfigSetting(key, stream.OutputTypeInMemory),
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

	var err error
	if _, msg.Attributes, err = s.encoder.Decode(context.Background(), msg, output); err != nil {
		s.failNow(err.Error(), "can not unmarshal message body")
	}

	return msg.Attributes
}

func (s *streamOutputComponent) UnmarshalAggregate(i int, output interface{}) []map[string]interface{} {
	msg, ok := s.Get(i)

	if !ok {
		s.failNow("message not available", "there is no message with index %d", i)
	}

	if msg.Attributes[stream.AttributeAggregate] != true {
		s.failNow("message not an aggregate", "there is no valid aggregate attribute on message with index %d", i)
	}

	batch := make([]*stream.Message, 0)
	s.Unmarshal(i, &batch)

	outputType := reflect.TypeOf(output)
	if outputType.Kind() != reflect.Ptr || outputType.Elem().Kind() != reflect.Slice || outputType.Elem().Elem().Kind() != reflect.Ptr {
		s.failNow("invalid output type", "can not unmarshal into %T, expected a *[]*TYPE", output)
	}

	sliceType := outputType.Elem()
	elemType := sliceType.Elem().Elem()
	slice := reflect.MakeSlice(sliceType, 0, len(batch))
	attributes := make([]map[string]interface{}, 0, len(batch))

	for _, msg := range batch {
		elem := reflect.New(elemType).Interface()

		var err error
		if _, msg.Attributes, err = s.encoder.Decode(context.Background(), msg, elem); err != nil {
			s.failNow(err.Error(), "can not unmarshal message body")
		}

		slice = reflect.Append(slice, reflect.ValueOf(elem))
		attributes = append(attributes, msg.Attributes)
	}

	reflect.ValueOf(output).Elem().Set(slice)

	return attributes
}
