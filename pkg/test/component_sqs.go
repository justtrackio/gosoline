package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/aws/aws-sdk-go/service/sqs"
	"log"
	"sync"
	"time"
)

type sqsConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

var sqsConfigs map[string]*sqsConfig
var sqsClients map[string]*sqs.SQS
var sqsLck sync.Mutex

func init() {
	sqsConfigs = map[string]*sqsConfig{}
	sqsClients = map[string]*sqs.SQS{}
}

func ProvideSqsClient(name string) *sqs.SQS {
	sqsLck.Lock()
	defer sqsLck.Unlock()

	_, ok := sqsClients[name]
	if ok {
		return sqsClients[name]
	}

	sess, err := getSession(sqsConfigs[name].Host, sqsConfigs[name].Port)

	if err != nil {
		logErr(err, "could not create sqs client: %s")
	}

	sqsClients[name] = sqs.New(sess)

	return sqsClients[name]
}

func runSqs(name string, config configInput) {
	wait.Add(1)
	go doRunSqs(name, config)
}

func doRunSqs(name string, configMap configInput) {
	defer wait.Done()
	defer log.Printf("%s component of type %s is ready", name, "sqs")

	localConfig := &sqsConfig{}
	unmarshalConfig(configMap, localConfig)
	sqsConfigs[name] = localConfig

	runContainer("gosoline_test_sqs", ContainerConfig{
		Repository: "localstack/localstack",
		Tag:        "0.10.7",
		Env: []string{
			"SERVICES=sqs",
		},
		PortBindings: PortBinding{
			"4576/tcp": fmt.Sprint(localConfig.Port),
		},
		HealthCheck: sqsHealthcheck(name),
	})
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
