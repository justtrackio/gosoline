package json

import "encoding/json"

func Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func MarshalIndent(v any, prefix string, indent string) ([]byte, error) {
	return json.MarshalIndent(v, prefix, indent)
}

func Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func Valid(data []byte) bool {
	return json.Valid(data)
}

type RawMessage = json.RawMessage

type Marshaler interface {
	json.Marshaler
}

type Unmarshaler interface {
	json.Unmarshaler
}
