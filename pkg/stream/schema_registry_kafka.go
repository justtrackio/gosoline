package stream

import (
	"context"
	"fmt"

	schemaRegistry "github.com/justtrackio/gosoline/pkg/kafka/schema-registry"
	"github.com/twmb/franz-go/pkg/sr"
)

var encodingToKafkaSchemaTypeMap = map[EncodingType]schemaRegistry.SchemaType{
	EncodingAvro:     schemaRegistry.Avro,
	EncodingJson:     schemaRegistry.Json,
	EncodingProtobuf: schemaRegistry.Protobuf,
}

func InitKafkaSchemaRegistry(
	ctx context.Context,
	settings SchemaSettingsWithEncoding,
	schemaRegistryService schemaRegistry.Service,
) (MessageBodyEncoder, error) {
	schemaType, ok := encodingToKafkaSchemaTypeMap[settings.Encoding]
	if !ok {
		return nil, fmt.Errorf("encoding %s is not supported by schema registry", settings.Encoding)
	}

	var encodeFn, decodeFn sr.EncodingOpt
	options := make([]sr.EncodingOpt, 0)

	switch schemaType {
	case schemaRegistry.Avro:
		avroEncoder, err := NewAvroEncoder(settings.Schema)
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
	case schemaRegistry.Json:
		jsonEncoder := NewJsonEncoder()

		encodeFn = sr.EncodeFn(func(v any) ([]byte, error) {
			return jsonEncoder.Encode(v)
		})
		decodeFn = sr.DecodeFn(func(b []byte, v any) error {
			return jsonEncoder.Decode(b, v)
		})

		options = append(options, encodeFn, decodeFn)
	case schemaRegistry.Protobuf:
		protoEncoder := NewProtobufEncoder()

		encodeFn = sr.EncodeFn(func(v any) ([]byte, error) {
			return protoEncoder.Encode(v)
		})
		decodeFn = sr.DecodeFn(func(b []byte, v any) error {
			return protoEncoder.Decode(b, v)
		})

		index := sr.Index(0)
		if len(settings.ProtobufMessageIndex) > 0 {
			index = sr.Index(settings.ProtobufMessageIndex...)
		}

		options = append(options, encodeFn, decodeFn, index)
	default:
		return nil, fmt.Errorf("unknown schema type: %s", schemaType)
	}

	schemaId, err := schemaRegistryService.GetSubjectSchemaId(ctx, settings.Subject, settings.Schema, schemaType)
	if err != nil {
		return nil, fmt.Errorf("failed to get subject schema id from registry: %w", err)
	}

	serde := schemaRegistry.NewSerde()
	serde.Register(schemaId, settings.Model, options...)

	return serde, nil
}
