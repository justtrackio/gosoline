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

func getQueueNames(config cfg.Config) ([]string, error) {
	var ok bool
	var reader queueNameReader
	var inputs = readAllInputTypes(config)
	var queueNames = make([]string, 0)

	for inputName, typ := range inputs {
		if reader, ok = queueNameReaders[typ]; !ok {
			return nil, fmt.Errorf("input type should be SNS/SQS")
		}

		queueName := reader(config, inputName)
		queueNames = append(queueNames, queueName)
	}

	return queueNames, nil
}

func queueNameReaderSns(config cfg.Config, input string) string {
	inputSettings, _ := readSnsInputSettings(config, input)

	return sqs.QueueName(inputSettings)
}

func queueNameReaderSqs(config cfg.Config, input string) string {
	inputSettings := readSqsInputSettings(config, input)

	return sqs.QueueName(inputSettings)
}
