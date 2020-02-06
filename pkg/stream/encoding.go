package stream

import "github.com/applike/gosoline/pkg/encoding/json"

const EncodingJson = "application/json"

var defaultMessageBodyEncoding = EncodingJson

func WithDefaultMessageBodyEncoding(encoding string) {
	defaultMessageBodyEncoding = encoding
}

type MessageBodyEncoder interface {
	Encode(data interface{}) (string, error)
}

var messageBodyEncoders = map[string]MessageBodyEncoder{
	EncodingJson: new(jsonEncoder),
}

type jsonEncoder struct {
}

func (e jsonEncoder) Encode(data interface{}) (string, error) {
	bytes, err := json.Marshal(data)

	return string(bytes), err
}

func (e jsonEncoder) Decode(data string, out interface{}) error {
	return json.Unmarshal([]byte(data), out)
}
