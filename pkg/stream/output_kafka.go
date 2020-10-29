package stream

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type KafkaProducer interface {
	Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error
}

type KafkaOutputSettings struct {
	Topic string `cfg:"topic" validate:"required"`
}

type kafkaOutput struct {
	producer KafkaProducer
	settings *KafkaOutputSettings
}

func NewKafkaOutput(logger mon.Logger, settings *KafkaOutputSettings) *kafkaOutput {
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost",
	})

	if err != nil {
		logger.Fatalf(err, "can not create kafka producer for output")
	}

	return NewKafkaOutputWithInterfaces(producer, settings)
}

func NewKafkaOutputWithInterfaces(producer KafkaProducer, settings *KafkaOutputSettings) *kafkaOutput {
	return &kafkaOutput{
		producer: producer,
		settings: settings,
	}
}

func (k *kafkaOutput) WriteOne(ctx context.Context, msg WritableMessage) error {
	return k.Write(ctx, []WritableMessage{msg})
}

func (k *kafkaOutput) Write(_ context.Context, batch []WritableMessage) error {
	var ok bool
	var err error
	var bytes []byte
	var deliveryCh = make(chan kafka.Event)
	var resEvent kafka.Event
	var resMsg *kafka.Message

	for _, msg := range batch {
		if bytes, err = msg.MarshalToBytes(); err != nil {
			return fmt.Errorf("can not marshal message to bytes: %w", err)
		}

		err = k.producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &k.settings.Topic, Partition: kafka.PartitionAny},
			Value:          bytes,
		}, deliveryCh)

		if err != nil {
			return fmt.Errorf("can not produce message: %w", err)
		}

		resEvent = <-deliveryCh

		if resMsg, ok = resEvent.(*kafka.Message); !ok {
			return fmt.Errorf("expected result *kafka.Message but got %T: %s", resEvent, resEvent.String())
		}

		if resMsg.TopicPartition.Error != nil {
			return fmt.Errorf("error on message delivery: %w", err)
		}
	}

	return nil
}
