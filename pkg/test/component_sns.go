package test

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/sns"
	"log"
	"strings"
	"sync"
)

type snsConfig struct {
	SqsEndpoint    string   `mapstructure:"sqs_endpoint"`
	LambdaEndpoint string   `mapstructure:"lambda_endpoint"`
	Services       []string `mapstructure:"services"`
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

	sess, err := getSession(snsConfigs[name].Port)

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

	runContainer("gosoline-test-sns", ContainerConfig{
		Repository: "localstack/localstack",
		Tag:        "0.10.3",
		Env:        envVariables,
		PortBindings: PortBinding{
			"4575/tcp": fmt.Sprint(localConfig.Port),
		},
		HealthCheck: func() error {
			snsClient := ProvideSnsClient(name)
			_, err := snsClient.ListTopics(&sns.ListTopicsInput{})

			return err
		},
	})
}
