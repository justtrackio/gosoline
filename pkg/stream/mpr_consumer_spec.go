package stream

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/sqs"
)

type ConsumerSpec struct {
	QueueName   string
	RunnerCount int
}

func GetConsumerSpecs(config cfg.Config, consumers []string) ([]*ConsumerSpec, error) {
	var err error
	var specs = make([]*ConsumerSpec, len(consumers))

	for i, consumer := range consumers {
		if specs[i], err = GetConsumerSpec(config, consumer); err != nil {
			return nil, fmt.Errorf("can't get consumer %s spec: %w", consumer, err)
		}
	}

	return specs, nil
}

func GetConsumerSpec(config cfg.Config, consumer string) (*ConsumerSpec, error) {
	consumerSettings := readConsumerSettings(config, consumer)
	inputType := readInputType(config, consumerSettings.Input)

	if inputType != InputTypeSqs {
		return nil, fmt.Errorf("input type is not SQS")
	}

	inputSettings := readSqsInputSettings(config, consumerSettings.Input)

	spec := &ConsumerSpec{
		QueueName:   sqs.QueueName(inputSettings),
		RunnerCount: consumerSettings.RunnerCount,
	}

	return spec, nil
}
