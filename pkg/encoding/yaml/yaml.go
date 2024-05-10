package yaml

import "gopkg.in/yaml.v3"

func Marshal(v any) ([]byte, error) {
	return yaml.Marshal(v)
}

func Unmarshal(data []byte, v any) error {
	return yaml.Unmarshal(data, v)
}
