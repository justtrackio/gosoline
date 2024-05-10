package stream

import "github.com/justtrackio/gosoline/pkg/encoding/json"

type jsonEncoder struct{}

func NewJsonEncoder() MessageBodyEncoder {
	return jsonEncoder{}
}

func (e jsonEncoder) Encode(data any) ([]byte, error) {
	return json.Marshal(data)
}

func (e jsonEncoder) Decode(data []byte, out any) error {
	return json.Unmarshal(data, out)
}
