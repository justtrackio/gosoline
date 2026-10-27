package stream

import (
	"fmt"

	"github.com/iskorotkov/avro/v2"
)

type avroEncoder struct {
	schema avro.Schema
	codec  avro.API
}

func NewAvroEncoder(schema string) (MessageBodyEncoder, error) {
	return newAvroEncoder(schema, nil)
}

// NewAvroEncoderWithCodec uses codec for both encoding and decoding.
// Register custom types and converters on the supplied codec before using the encoder.
func NewAvroEncoderWithCodec(schema string, codec avro.API) (MessageBodyEncoder, error) {
	if codec == nil {
		return nil, fmt.Errorf("avro codec must not be nil")
	}

	return newAvroEncoder(schema, codec)
}

func newAvroEncoder(schema string, codec avro.API) (MessageBodyEncoder, error) {
	avroSchema, err := avro.Parse(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to parse avro schema: %w", err)
	}

	return avroEncoder{
		schema: avroSchema,
		codec:  codec,
	}, nil
}

func (e avroEncoder) Encode(data any) ([]byte, error) {
	if e.codec != nil {
		return e.codec.Marshal(e.schema, data)
	}

	return avro.Marshal(e.schema, data)
}

func (e avroEncoder) Decode(data []byte, out any) error {
	if e.codec != nil {
		return e.codec.Unmarshal(e.schema, data, out)
	}

	return avro.Unmarshal(e.schema, data, out)
}
