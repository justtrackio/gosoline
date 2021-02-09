package stream

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/sqs"
)

type queueNameReader func(config cfg.Config, input string) string

var queueNameReaders = map[string]queueNameReader{
	InputTypeSqs: queueNameReaderSqs,
	InputTypeSns: queueNameReaderSns,
}

type ConsumerSpec struct {
	ConsumerName string
	QueueName    string
}

func GetConsumerSpecs(config cfg.Config) ([]*ConsumerSpec, error) {
	var err error
	var spec *ConsumerSpec
	var specs = make([]*ConsumerSpec, 0)
	var consumers = readAllConsumerNames(config)

	for _, consumer := range consumers {
		if spec, err = GetConsumerSpec(config, consumer); err != nil {
			return nil, fmt.Errorf("can't get consumer %s spec: %w", consumer, err)
		}

		specs = append(specs, spec)
	}

	return specs, nil
}

func GetConsumerSpec(config cfg.Config, consumer string) (*ConsumerSpec, error) {
	consumerSettings := readConsumerSettings(config, consumer)
	inputType := readInputType(config, consumerSettings.Input)

	var ok bool
	var reader queueNameReader

	if reader, ok = queueNameReaders[inputType]; !ok {
		return nil, fmt.Errorf("input type should be SNS/SQS")
	}

	queueName := reader(config, consumerSettings.Input)

	spec := &ConsumerSpec{
		ConsumerName: consumer,
		QueueName:    queueName,
	}

	return spec, nil
}

func queueNameReaderSns(config cfg.Config, input string) string {
	inputSettings, _ := readSnsInputSettings(config, input)

	return sqs.QueueName(inputSettings)
}

func queueNameReaderSqs(config cfg.Config, input string) string {
	inputSettings := readSqsInputSettings(config, input)

	return sqs.QueueName(inputSettings)
}
