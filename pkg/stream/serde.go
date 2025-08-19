package stream

//go:generate go run github.com/vektra/mockery/v2 --name Serde
type Serde interface {
	Decode(b []byte, v any) error
	Encode(v any) ([]byte, error)
}
