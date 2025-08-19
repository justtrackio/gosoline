package stream

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kafka/connection"
	kafkaProducer "github.com/justtrackio/gosoline/pkg/kafka/producer"
	schemaRegistry "github.com/justtrackio/gosoline/pkg/kafka/schema-registry"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/twmb/franz-go/pkg/sr"
)

type kafkaOutput struct {
	connection            connection.Settings
	schemaRegistryService schemaRegistry.Service
	writer                kafkaProducer.Writer
}

var (
	_ SizeRestrictedOutput      = &kafkaOutput{}
	_ SchemaRegistryAwareOutput = &kafkaOutput{}
)

func NewKafkaOutput(ctx context.Context, config cfg.Config, logger log.Logger, settings *kafkaProducer.Settings) (Output, error) {
	writer, err := kafkaProducer.NewWriter(ctx, config, logger, settings)
	if err != nil {
		return nil, fmt.Errorf("can not create kafka writer: %w", err)
	}

	conn, err := connection.ParseSettings(config, settings.Connection)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kafka connection settings for connection name %q: %w", settings.Connection, err)
	}

	service, err := schemaRegistry.NewService(*conn)
	if err != nil {
		return nil, fmt.Errorf("can not create schema registry service: %w", err)
	}

	return NewKafkaOutputWithInterfaces(*conn, service, writer)
}

func NewKafkaOutputWithInterfaces(
	connection connection.Settings,
	schemaRegistryService schemaRegistry.Service,
	writer kafkaProducer.Writer,
) (Output, error) {
	return &kafkaOutput{
		connection:            connection,
		schemaRegistryService: schemaRegistryService,
		writer:                writer,
	}, nil
}

func (o *kafkaOutput) GetSerde(ctx context.Context, settings SchemaSettingsWithEncoding) (Serde, error) {
	if o.connection.SchemaRegistryAddress == "" {
		return nil, fmt.Errorf("no schema registry address provided")
	}

	schemaType, ok := encodingToSchemaTypeMap[settings.Encoding]
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

	schemaId, err := o.schemaRegistryService.GetSubjectSchemaId(ctx, settings.Subject, settings.Schema, schemaType)
	if err != nil {
		return nil, fmt.Errorf("failed to get subject schema id from registry: %w", err)
	}

	serde := schemaRegistry.NewSerde()
	serde.Register(schemaId, settings.Model, options...)

	return serde, nil
}

func (o *kafkaOutput) WriteOne(ctx context.Context, m WritableMessage) error {
	message, err := NewKafkaMessage(m)
	if err != nil {
		return fmt.Errorf("failed to build kafka message: %w", err)
	}

	return o.writer.ProduceSync(ctx, message).FirstErr()
}

func (o *kafkaOutput) Write(ctx context.Context, ms []WritableMessage) error {
	messages, err := NewKafkaMessages(ms)
	if err != nil {
		return fmt.Errorf("failed to build kafka messages: %w", err)
	}

	return o.writer.ProduceSync(ctx, messages...).FirstErr()
}

func (o *kafkaOutput) ProvidesCompression() bool {
	return true
}

func (o *kafkaOutput) SupportsAggregation() bool {
	return false
}

func (o *kafkaOutput) IsPartitionedOutput() bool {
	// we are not using the partitioned producer daemon aggregator.
	// but the kafka library will partition by the AttributeKafkaKey in the message attributes if it is set.
	return false
}

func (o *kafkaOutput) GetMaxMessageSize() *int {
	return mdl.Box(1000 * 1000)
}

func (o *kafkaOutput) GetMaxBatchSize() *int {
	return mdl.Box(500)
}
