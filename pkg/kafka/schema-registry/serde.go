package schema_registry

import (
	"github.com/twmb/franz-go/pkg/sr"
)

//go:generate go run github.com/vektra/mockery/v2 --name Serde
type Serde interface {
	Decode(b []byte, attrs map[string]string, v any) error
	Encode(v any, attrs map[string]string) ([]byte, error)
	Register(id int, v any, opts ...sr.EncodingOpt)
}

type SerdeWrapper struct {
	serde sr.Serde
}

func (s *SerdeWrapper) Decode(b []byte, _ map[string]string, v any) error {
	return s.serde.Decode(b, v)
}

func (s *SerdeWrapper) Encode(v any, _ map[string]string) ([]byte, error) {
	return s.serde.Encode(v)
}

func (s *SerdeWrapper) Register(id int, v any, opts ...sr.EncodingOpt) {
	s.serde.Register(id, v, opts...)
}

func NewSerde() Serde {
	return &SerdeWrapper{serde: sr.Serde{}}
}
