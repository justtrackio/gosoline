package stream

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/kafka/errors"
	schemaRegistry "github.com/justtrackio/gosoline/pkg/kafka/schema-registry"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/twmb/franz-go/pkg/sr"
)

var encodingToKafkaSchemaTypeMap = map[EncodingType]schemaRegistry.SchemaType{
	EncodingAvro:     schemaRegistry.Avro,
	EncodingJson:     schemaRegistry.Json,
	EncodingProtobuf: schemaRegistry.Protobuf,
}

func InitKafkaSchemaRegistry(
	ctx context.Context,
	logger log.Logger,
	settings SchemaSettingsWithEncoding,
	backoff exec.BackoffSettings,
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

	getSchemaId := schemaRegistryService.GetSubjectSchemaId
	if settings.AutoRegister {
		getSchemaId = schemaRegistryService.GetOrCreateSubjectSchemaId
	}

	executor := exec.NewBackoffExecutor(logger, &exec.ExecutableResource{
		Type: "schema-registry",
		Name: settings.Subject,
	}, &backoff, []exec.ErrorChecker{
		func(result any, err error) exec.ErrorType {
			if errors.IsRetryableKafkaError(err) {
				return exec.ErrorTypeRetryable
			}

			return exec.ErrorTypeUnknown
		},
	})

	result, err := executor.Execute(ctx, func(ctx context.Context) (any, error) {
		return getSchemaId(ctx, settings.Subject, settings.Schema, schemaType)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get subject schema id from registry: %w", err)
	}

	schemaId := result.(int)

	serde := schemaRegistry.NewSerde()
	serde.Register(schemaId, settings.Model, options...)

	return serde, nil
}
