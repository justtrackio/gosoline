package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/aws/aws-sdk-go/service/sns"
	"log"
	"strings"
	"sync"
	"time"
)

type snsConfig struct {
	SqsEndpoint    string   `mapstructure:"sqs_endpoint"`
	LambdaEndpoint string   `mapstructure:"lambda_endpoint"`
	Services       []string `mapstructure:"services"`
	Host           string   `mapstructure:"host"`
	Port           int      `mapstructure:"port"`
}

var snsClients map[string]*sns.SNS
var snsConfigs map[string]*snsConfig
var snsLck sync.Mutex

func init() {
	snsConfigs = map[string]*snsConfig{}
	snsClients = map[string]*sns.SNS{}
}

func ProvideSnsClient(name string) *sns.SNS {
	snsLck.Lock()
	defer snsLck.Unlock()

	_, ok := snsClients[name]
	if ok {
		return snsClients[name]
	}

	sess, err := getSession(snsConfigs[name].Host, snsConfigs[name].Port)

	if err != nil {
		logErr(err, "could not create sns client: %s")
	}

	snsClients[name] = sns.New(sess)

	return snsClients[name]
}

func runSns(name string, config configInput) {
	wait.Add(1)
	go doRunSns(name, config)
}

func doRunSns(name string, configMap configInput) {
	defer wait.Done()
	defer log.Printf("%s component of type %s is ready", name, "sns")

	localConfig := &snsConfig{}
	unmarshalConfig(configMap, localConfig)
	snsConfigs[name] = localConfig

	services := []string{
		"sns",
	}

	envVariables := make([]string, 0)

	if len(localConfig.LambdaEndpoint) > 0 {
		envVariables = append(envVariables, "LAMBDA_BACKEND="+localConfig.LambdaEndpoint)
		services = append(services, "lambda")
	}

	if len(localConfig.SqsEndpoint) > 0 {
		envVariables = append(envVariables, "SQS_BACKEND="+localConfig.SqsEndpoint)
		services = append(services, "sqs")
	}

	envVariables = append(envVariables, "SERVICES="+strings.Join(services, ","))

	runContainer("gosoline_test_sns", ContainerConfig{
		Repository: "localstack/localstack",
		Tag:        "0.10.7",
		Env:        envVariables,
		PortBindings: PortBinding{
			"4575/tcp": fmt.Sprint(localConfig.Port),
		},
		HealthCheck: snsHealthcheck(name),
	})
}

func snsHealthcheck(name string) func() error {
	return func() error {
		snsClient := ProvideSnsClient(name)
		topicName := "healthcheck"

		topic, err := snsClient.CreateTopic(&sns.CreateTopicInput{
			Name: mdl.String(topicName),
		})

		if err != nil {
			return err
		}

		listTopics, err := snsClient.ListTopics(&sns.ListTopicsInput{})

		if err != nil {
			return err
		}

		if len(listTopics.Topics) != 1 {
			return fmt.Errorf("topic list should contain exactly 1 entry, but contained %d", len(listTopics.Topics))
		}

		_, err = snsClient.DeleteTopic(&sns.DeleteTopicInput{TopicArn: topic.TopicArn})

		if err != nil {
			return err
		}

		// wait for topic to be really deleted (race condition)
		for {
			listTopics, err := snsClient.ListTopics(&sns.ListTopicsInput{})

			if err != nil {
				return err
			}

			if len(listTopics.Topics) == 0 {
				return nil
			}

			time.Sleep(50 * time.Millisecond)
		}
	}
}
