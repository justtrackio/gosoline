package stream

import (
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/sqs"
)

type queueNameReader func(config cfg.Config, input string) string

var queueNameReaders = map[string]queueNameReader{
	InputTypeSqs: queueNameReaderSqs,
	InputTypeSns: queueNameReaderSns,
}

func getQueueNames(config cfg.Config) ([]string, error) {
	var ok bool
	var reader queueNameReader

	inputs := readAllInputTypes(config)
	queueNames := make([]string, 0)

	for inputName, typ := range inputs {
		if reader, ok = queueNameReaders[typ]; !ok {
			// it is an input we can't measure (e.g., an inMemory input), skip it as there is no useful metric for us to write
			continue
		}

		queueName := reader(config, inputName)
		queueNames = append(queueNames, queueName)
	}

	return queueNames, nil
}

func queueNameReaderSns(config cfg.Config, input string) string {
	inputSettings, _ := readSnsInputSettings(config, input)

	return sqs.GetQueueName(inputSettings)
}

func queueNameReaderSqs(config cfg.Config, input string) string {
	inputSettings := readSqsInputSettings(config, input)

	return sqs.GetQueueName(inputSettings)
}
