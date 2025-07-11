package stream

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/sqs"
)

type queueNameReader func(config cfg.Config, input string) (string, error)

var queueNameReaders = map[string]queueNameReader{
	InputTypeSqs: queueNameReaderSqs,
	InputTypeSns: queueNameReaderSns,
}

func getQueueNames(config cfg.Config) ([]string, error) {
	var ok bool
	var err error
	var queueName string
	var reader queueNameReader

	inputs, err := readAllInputTypes(config)
	if err != nil {
		return nil, fmt.Errorf("failed to read input types: %w", err)
	}
	queueNames := make([]string, 0)

	for inputName, typ := range inputs {
		if reader, ok = queueNameReaders[typ]; !ok {
			// it is an input we can't measure (e.g., an inMemory input), skip it as there is no useful metric for us to write
			continue
		}

		if queueName, err = reader(config, inputName); err != nil {
			return nil, fmt.Errorf("can not get queue name for input %s: %w", inputName, err)
		}

		queueNames = append(queueNames, queueName)
	}

	return queueNames, nil
}

func queueNameReaderSns(config cfg.Config, input string) (string, error) {
	inputSettings, _ := readSnsInputSettings(config, input)

	return sqs.GetQueueName(config, inputSettings)
}

func queueNameReaderSqs(config cfg.Config, input string) (string, error) {
	inputSettings := readSqsInputSettings(config, input)

	return sqs.GetQueueName(config, inputSettings)
}
