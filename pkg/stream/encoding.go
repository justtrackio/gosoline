package stream

import (
	"fmt"
	"github.com/applike/gosoline/pkg/encoding/json"
	"github.com/spf13/cast"
)

const (
	EncodingJson = "application/json"
	EncodingText = "text/plain"
)

var defaultMessageBodyEncoding = EncodingJson

func WithDefaultMessageBodyEncoding(encoding string) {
	defaultMessageBodyEncoding = encoding
}

type MessageBodyEncoder interface {
	Encode(data interface{}) ([]byte, error)
	Decode(data []byte, out interface{}) error
}

var messageBodyEncoders = map[string]MessageBodyEncoder{
	EncodingJson: new(jsonEncoder),
	EncodingText: new(textEncoder),
}

func AddMessageBodyEncoder(encoding string, encoder MessageBodyEncoder) {
	messageBodyEncoders[encoding] = encoder
}

type jsonEncoder struct{}

func (e jsonEncoder) Encode(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}

func (e jsonEncoder) Decode(data []byte, out interface{}) error {
	if _, ok := out.(*[]byte); ok {
		return messageBodyEncoders[EncodingText].Decode(data, out)
	}

	return json.Unmarshal(data, out)
}

type textEncoder struct{}

func (e textEncoder) Encode(data interface{}) ([]byte, error) {
	if bts, ok := data.([]byte); ok {
		return bts, nil
	}

	str, err := cast.ToStringE(data)

	return []byte(str), err
}

func (e textEncoder) Decode(data []byte, out interface{}) error {
	bts, ok := out.(*[]byte)
	if !ok {
		return fmt.Errorf("the out parameter of the text decode has to be a pointer to byte slice")
	}

	*bts = append(*bts, data...)

	return nil
}
