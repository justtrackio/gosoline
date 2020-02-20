package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/aws/aws-sdk-go/service/sqs"
	"sync"
	"time"
)

var sqsClients map[string]*sqs.SQS
var sqsLck sync.Mutex

func init() {
	sqsClients = map[string]*sqs.SQS{}
}

func ProvideSqsClient(name string) *sqs.SQS {
	sqsLck.Lock()
	defer sqsLck.Unlock()

	_, ok := sqsClients[name]
	if ok {
		return sqsClients[name]
	}

	sess, err := getSession(snsSqsConfigs[name].Host, snsSqsConfigs[name].SqsPort)

	if err != nil {
		logErr(err, "could not create sqs client: %s")
	}

	sqsClients[name] = sqs.New(sess)

	return sqsClients[name]
}

func sqsHealthcheck(name string) func() error {
	return func() error {
		sqsClient := ProvideSqsClient(name)
		queueName := "healthcheck"

		_, err = sqsClient.CreateQueue(&sqs.CreateQueueInput{
			QueueName: mdl.String(queueName),
		})

		if err != nil {
			return err
		}

		listQueues, err := sqsClient.ListQueues(&sqs.ListQueuesInput{})

		if err != nil {
			return err
		}

		if len(listQueues.QueueUrls) != 1 {
			return fmt.Errorf("queue  list should contain exactly 1 entry, but contained %d", len(listQueues.QueueUrls))
		}

		_, err = sqsClient.DeleteQueue(&sqs.DeleteQueueInput{QueueUrl: listQueues.QueueUrls[0]})

		if err != nil {
			return err
		}

		// wait for queue to be really deleted (race condition)
		for {
			listQueues, err := sqsClient.ListQueues(&sqs.ListQueuesInput{QueueNamePrefix: mdl.String(queueName)})

			if err != nil {
				return err
			}

			if len(listQueues.QueueUrls) == 0 {
				return nil
			}

			time.Sleep(50 * time.Millisecond)
		}
	}
}
