package stream

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/sqs"
	"regexp"
)

type ConsumerSpec struct {
	ConsumerName string
	QueueName    string
	RunnerCount  int
}

func GetConsumerSpecs(config cfg.Config, patterns []string) ([]*ConsumerSpec, error) {
	var err error
	var matcher *regexp.Regexp
	var spec *ConsumerSpec
	var specs = make([]*ConsumerSpec, 0)
	var consumers = readAllConsumerNames(config)

	for _, pattern := range patterns {
		if matcher, err = regexp.Compile("^" + pattern + "$"); err != nil {
			return nil, fmt.Errorf("%s is not a valid regexp pattern: %w", pattern, err)
		}

		for _, consumer := range consumers {
			if !matcher.MatchString(consumer) {
				continue
			}

			if spec, err = GetConsumerSpec(config, consumer); err != nil {
				return nil, fmt.Errorf("can't get consumer %s spec: %w", consumer, err)
			}

			specs = append(specs, spec)
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
		ConsumerName: consumer,
		QueueName:    sqs.QueueName(inputSettings),
		RunnerCount:  consumerSettings.RunnerCount,
	}

	return spec, nil
}
