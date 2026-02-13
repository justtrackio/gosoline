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

type avroSchemaCarrierEncoder struct{}

type SchemaCarrier interface {
	Schema() avro.Schema
}

func (e avroSchemaCarrierEncoder) Encode(data any) ([]byte, error) {
	avroData, ok := data.(SchemaCarrier)
	if !ok {
		return nil, fmt.Errorf("%T does not implement SchemaCarrier", data)
	}

	return avro.Marshal(avroData.Schema(), data)
}

func (e avroSchemaCarrierEncoder) Decode(data []byte, out any) error {
	avroData, ok := out.(SchemaCarrier)
	if !ok {
		return fmt.Errorf("%T does not implement SchemaCarrier", data)
	}

	return avro.Unmarshal(avroData.Schema(), data, out)
}
