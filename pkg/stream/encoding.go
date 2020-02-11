package stream

import (
	"fmt"
	"github.com/applike/gosoline/pkg/encoding/json"
	"github.com/spf13/cast"
)

const EncodingJson = "application/json"
const EncodingText = "text/plain"

var defaultMessageBodyEncoding = EncodingJson

func WithDefaultMessageBodyEncoding(encoding string) {
	defaultMessageBodyEncoding = encoding
}

type MessageBodyEncoder interface {
	Encode(data interface{}) (string, error)
	Decode(data string, out interface{}) error
}

var messageBodyEncoders = map[string]MessageBodyEncoder{
	EncodingJson: new(jsonEncoder),
	EncodingText: new(textEncoder),
}

type jsonEncoder struct{}

func (e jsonEncoder) Encode(data interface{}) (string, error) {
	bytes, err := json.Marshal(data)

	return string(bytes), err
}

func (e jsonEncoder) Decode(data string, out interface{}) error {
	return json.Unmarshal([]byte(data), out)
}

type textEncoder struct{}

func (e textEncoder) Encode(data interface{}) (string, error) {
	if str, ok := data.(string); ok {
		return str, nil
	}

	return cast.ToStringE(data)
}

func (e textEncoder) Decode(data string, out interface{}) error {
	if ptr, ok := out.(*string); ok {
		*ptr = data
		return nil
	}

	return fmt.Errorf("the out parameter of the text decode has to be a pointer to string")
}
