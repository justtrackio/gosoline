package schema_registry

import (
	"github.com/twmb/franz-go/pkg/sr"
)

//go:generate go run github.com/vektra/mockery/v2 --name Serde
type Serde interface {
	Decode(b []byte, v any) error
	Encode(v any) ([]byte, error)
	Register(id int, v any, opts ...sr.EncodingOpt)
}

func NewSerde() Serde {
	var serde sr.Serde

	return &serde
}
