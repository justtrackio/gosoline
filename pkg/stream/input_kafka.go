package stream

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/kafka/connection"
	kafkaConsumer "github.com/justtrackio/gosoline/pkg/kafka/consumer"
	schemaRegistry "github.com/justtrackio/gosoline/pkg/kafka/schema-registry"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sr"
)

type kafkaInput struct {
	logger                log.Logger
	connection            connection.Settings
	healthCheckTimer      clock.HealthCheckTimer
	polling               atomic.Bool
	partitionManager      kafkaConsumer.PartitionManager
	reader                kafkaConsumer.Reader
	schemaRegistryService schemaRegistry.Service
	maxPollRecords        int
	data                  chan *Message
}

var _ SchemaRegistryAwareInput = &kafkaInput{}

func NewKafkaInput(ctx context.Context, config cfg.Config, logger log.Logger, settings kafkaConsumer.Settings) (Input, error) {
	data := make(chan *Message)
	messageHandler := NewKafkaMessageHandler(data)
	partitionManager := kafkaConsumer.NewPartitionManager(logger, messageHandler)

	opts := []kgo.Opt{
		kgo.OnPartitionsAssigned(partitionManager.OnPartitionsAssigned),
		kgo.OnPartitionsRevoked(partitionManager.OnPartitionsLostOrRevoked),
		kgo.OnPartitionsLost(partitionManager.OnPartitionsLostOrRevoked),
	}

	reader, err := kafkaConsumer.NewReader(ctx, config, logger, settings, opts...)
	if err != nil {
		return nil, fmt.Errorf("can not create kafka reader: %w", err)
	}

	conn, err := connection.ParseSettings(config, settings.Connection)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kafka connection settings for connection name %q: %w", settings.Connection, err)
	}

	service, err := schemaRegistry.NewService(*conn)
	if err != nil {
		return nil, fmt.Errorf("can not create schema registry service: %w", err)
	}

	healthCheckTimer, err := clock.NewHealthCheckTimer(settings.Healthcheck.Timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to create healthcheck timer: %w", err)
	}

	return NewKafkaInputWithInterfaces(logger, *conn, healthCheckTimer, *partitionManager, reader, service, settings.MaxPollRecords, data)
}

func NewKafkaInputWithInterfaces(
	logger log.Logger,
	connection connection.Settings,
	healthCheckTimer clock.HealthCheckTimer,
	partitionManager kafkaConsumer.PartitionManager,
	reader kafkaConsumer.Reader,
	schemaRegistryService schemaRegistry.Service,
	maxPollRecords int,
	data chan *Message,
) (Input, error) {
	return &kafkaInput{
		logger:                logger,
		connection:            connection,
		healthCheckTimer:      healthCheckTimer,
		partitionManager:      partitionManager,
		reader:                reader,
		schemaRegistryService: schemaRegistryService,
		maxPollRecords:        maxPollRecords,
		data:                  data,
	}, nil
}

func (i *kafkaInput) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// while we are polling messages, we can't get unhealthy
		// (as this code is outside our control to add code to mark us as healthy)
		i.polling.Store(true)
		fetches := i.reader.PollRecords(ctx, i.maxPollRecords)
		// mark us as healthy as soon as we got records to ensure we stay healthy while we process the records
		// (unless we take too long to send the messages to the i.data channel)
		i.healthCheckTimer.MarkHealthy()
		i.polling.Store(false)

		if fetches.IsClientClosed() {
			return nil
		}
		if errors.Is(fetches.Err0(), context.Canceled) {
			return ctx.Err()
		}

		fetches.EachError(func(topic string, partition int32, err error) {
			var errDataLoss *kgo.ErrDataLoss

			switch {
			case errors.As(err, &errDataLoss):
				// the kafka library declares this error as informational (as it will reset and retry) but worth logging and investigating.
				// so, we log this as a warning.
				i.logger.WithContext(ctx).Warn("%s", err.Error())
			default:
				i.logger.WithContext(ctx).Error("failed to fetch records (topic: %s. partition: %d): %w", topic, partition, err)
			}
		})

		fetches.EachPartition(func(p kgo.FetchTopicPartition) {
			i.partitionManager.AssignRecords(p.Topic, p.Partition, p.Records)
		})

		i.reader.AllowRebalance()
	}
}

func (i *kafkaInput) Stop() {
	defer close(i.data)
	i.reader.CloseAllowingRebalance()
}

func (i *kafkaInput) Data() <-chan *Message {
	return i.data
}

func (i *kafkaInput) IsHealthy() bool {
	return i.healthCheckTimer.IsHealthy() || i.polling.Load()
}

func (i *kafkaInput) GetSerde(ctx context.Context, settings SchemaSettingsWithEncoding) (Serde, error) {
	if i.connection.SchemaRegistryAddress == "" {
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

	schemaId, err := i.schemaRegistryService.GetSubjectSchemaId(ctx, settings.Subject, settings.Schema, schemaType)
	if err != nil {
		return nil, fmt.Errorf("failed to get subject schema id from registry: %w", err)
	}

	serde := schemaRegistry.NewSerde()
	serde.Register(schemaId, settings.Model, options...)

	return serde, nil
}
