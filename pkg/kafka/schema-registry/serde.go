package schema_registry

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/twmb/franz-go/pkg/sr"
)

//go:generate go run github.com/vektra/mockery/v2 --name Serde
type Serde interface {
	Decode(b []byte, v any) error
	Encode(v any) ([]byte, error)
}

func NewSerde(schemaId int, schema string, schemaType SchemaType, model any) (Serde, error) {
	var serde sr.Serde
	var encodeFn, decodeFn sr.EncodingOpt

	options := make([]sr.EncodingOpt, 0)

	switch schemaType {
	case Avro:
		avroEncoder, err := stream.NewAvroEncoder(schema)
		if err != nil {
			return nil, fmt.Errorf("failed to create avro encoder: %w", err)
		}

		encodeFn = sr.EncodeFn(func(v any) ([]byte, error) {
			return avroEncoder.Encode(v)
		})
		decodeFn = sr.DecodeFn(func(b []byte, v any) error {
			return avroEncoder.Decode(b, v)
		})

		options = append(options, encodeFn, decodeFn)
	case Json:
		jsonEncoder := stream.NewJsonEncoder()

		encodeFn = sr.EncodeFn(func(v any) ([]byte, error) {
			return jsonEncoder.Encode(v)
		})
		decodeFn = sr.DecodeFn(func(b []byte, v any) error {
			return jsonEncoder.Decode(b, v)
		})

		options = append(options, encodeFn, decodeFn)
	case Protobuf:
		protoEncoder := stream.NewProtobufEncoder()

		encodeFn = sr.EncodeFn(func(v any) ([]byte, error) {
			return protoEncoder.Encode(v)
		})
		decodeFn = sr.DecodeFn(func(b []byte, v any) error {
			return protoEncoder.Decode(b, v)
		})

		index := sr.Index(0) // todo: need to somehow get the correct protobuf message index in here

		options = append(options, encodeFn, decodeFn, index)
	default:
		return nil, fmt.Errorf("unknown serde type: %s", schemaType)
	}

	serde.Register(schemaId, model, options...)

	return &serde, nil
}
