package stream

import "github.com/justtrackio/gosoline/pkg/encoding/json"

type jsonEncoder struct{}

func NewJsonEncoder() MessageBodyEncoder {
	return jsonEncoder{}
}

func (e jsonEncoder) Encode(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}

func (e jsonEncoder) Decode(data []byte, out interface{}) error {
	return json.Unmarshal(data, out)
}
