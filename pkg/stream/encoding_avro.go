package stream

import (
	"fmt"

	"github.com/hamba/avro/v2"
)

type avroEncoder struct {
	schema avro.Schema
}

func NewAvroEncoder(schema string) (MessageBodyEncoder, error) {
	avroSchema, err := avro.Parse(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to parse avro schema: %w", err)
	}

	return avroEncoder{
		schema: avroSchema,
	}, nil
}

func (e avroEncoder) Encode(data any) ([]byte, error) {
	return avro.Marshal(e.schema, data)
}

func (e avroEncoder) Decode(data []byte, out any) error {
	return avro.Unmarshal(e.schema, data, out)
}
