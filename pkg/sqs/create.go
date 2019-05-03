package sqs

import (
	"fmt"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"sync"
)

var cqLock sync.Mutex

func CreateQueue(logger mon.Logger, client sqsiface.SQSAPI, s Settings) {
	if !s.AutoCreate {
		return
	}

	cqLock.Lock()
	defer cqLock.Unlock()

	name := namingStrategy(s.AppId, s.QueueId)
	exists := QueueExists(logger, client, s)

	if exists {
		return
	}

	logger.Info(fmt.Sprintf("trying to create sqs queue: %v", name))

	input := &sqs.CreateQueueInput{
		QueueName: aws.String(name),
	}

	_, err := client.CreateQueue(input)

	if err != nil {
		logger.Fatal(err, fmt.Sprintf("could not create sqs queue %v", name))
	}

	logger.Info(fmt.Sprintf("created sqs queue %v", name))
}
