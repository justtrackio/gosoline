package test

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/sqs"
	"log"
	"sync"
)

type sqsConfig struct {
	Port int `mapstructure:"port"`
}

var sqsClients map[string]*sqs.SQS
var sqsConfigs map[string]*sqsConfig
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

	sess, err := getSession(sqsConfigs[name].Port)

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

	runContainer("gosoline-test-sqs", ContainerConfig{
		Repository: "localstack/localstack",
		Tag:        "0.10.3",
		Env: []string{
			"SERVICES=sqs",
		},
		PortBindings: PortBinding{
			"4576/tcp": fmt.Sprint(localConfig.Port),
		},
		HealthCheck: func() error {
			sqsClient := ProvideSqsClient(name)
			_, err := sqsClient.ListQueues(&sqs.ListQueuesInput{})

			return err
		},
	})
}
