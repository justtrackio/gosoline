package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/ory/dockertest/docker"
	"github.com/thoas/go-funk"
	"log"
	"strings"
	"sync"
	"time"
)

type snsSqsConfig struct {
	Host    string `mapstructure:"host"`
	SnsPort int    `mapstructure:"sns_port"`
	SqsPort int    `mapstructure:"sqs_port"`
}

var snsClients map[string]*sns.SNS
var snsSqsConfigs map[string]*snsSqsConfig
var snsLck sync.Mutex

func init() {
	snsSqsConfigs = map[string]*snsSqsConfig{}
	snsClients = map[string]*sns.SNS{}
}

func ProvideSnsClient(name string) *sns.SNS {
	snsLck.Lock()
	defer snsLck.Unlock()

	_, ok := snsClients[name]
	if ok {
		return snsClients[name]
	}

	sess, err := getSession(snsSqsConfigs[name].Host, snsSqsConfigs[name].SnsPort)

	if err != nil {
		logErr(err, "could not create sns client: %s")
	}

	snsClients[name] = sns.New(sess)

	return snsClients[name]
}

func runSnsSqs(name string, config configInput) {
	wait.Add(1)
	go doRunSnsSqs(name, config)
}

func doRunSnsSqs(name string, configMap configInput) {
	defer wait.Done()
	defer log.Printf("%s component of type %s is ready", name, "sns_sqs")

	localConfig := &snsSqsConfig{}
	unmarshalConfig(configMap, localConfig)
	snsSqsConfigs[name] = localConfig

	services := []string{
		"sns",
		"sqs",
	}

	envVariables := "SERVICES=" + strings.Join(services, ",")

	runContainer("gosoline_test_sns_sqs", ContainerConfig{
		Repository: "localstack/localstack",
		Tag:        "0.10.7",
		Env:        []string{envVariables},
		PortBindings: PortBinding{
			"4575/tcp": fmt.Sprint(localConfig.SnsPort),
			"4576/tcp": fmt.Sprint(localConfig.SqsPort),
		},
		HealthCheck: func() error {
			c, _ := dockerPool.Client.InspectContainer("gosoline_test_sns_sqs")

			funk.ForEach(c.NetworkSettings.Networks, func(_ string, elem docker.ContainerNetwork) {
				localConfig.Host = elem.IPAddress
				log.Println(fmt.Sprintf("set Host to %s", localConfig.Host))
			})

			err := snsHealthcheck(name)()

			if err != nil {
				return err
			}

			return sqsHealthcheck(name)()
		},
	})

	c, _ := dockerPool.Client.InspectContainer("gosoline_test_sns_sqs")

	funk.ForEach(c.NetworkSettings.Networks, func(_ string, elem docker.ContainerNetwork) {
		localConfig.Host = elem.IPAddress
		log.Println(fmt.Sprintf("set Host to %s", localConfig.Host))
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
