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

func (e avroEncoder) Encode(data any, _ map[string]string) ([]byte, error) {
	return avro.Marshal(e.schema, data)
}

func (e avroEncoder) Decode(data []byte, _ map[string]string, out any) error {
	return avro.Unmarshal(e.schema, data, out)
}
