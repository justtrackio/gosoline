package yaml

import "gopkg.in/yaml.v3"

func Marshal(v interface{}) ([]byte, error) {
	return yaml.Marshal(v)
}

func Unmarshal(data []byte, v interface{}) error {
	return yaml.Unmarshal(data, v)
}
