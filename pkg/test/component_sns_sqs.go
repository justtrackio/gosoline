package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"strings"
)

type snsSqsSettings struct {
	*mockSettings
	SnsPort int `cfg:"sns_port"`
	SqsPort int `cfg:"sqs_port"`
}

type snsSqsComponent struct {
	name     string
	settings *snsSqsSettings
	clients  *simpleCache
	runner   *dockerRunner
}

func (s *snsSqsComponent) Boot(config cfg.Config, runner *dockerRunner, settings *mockSettings, name string) {
	s.name = name
	s.runner = runner
	s.clients = &simpleCache{}
	s.settings = &snsSqsSettings{
		mockSettings: settings,
	}
	key := fmt.Sprintf("mocks.%s", name)
	config.UnmarshalKey(key, s.settings)
}

func (s *snsSqsComponent) Start() {
	services := "SERVICES=" + strings.Join([]string{
		"sns",
		"sqs",
	}, ",")

	env := []string{services}

	if s.settings.Debug {
		env = append(env, "DEBUG=1")
	}

	containerName := fmt.Sprintf("gosoline_test_sns_sqs_%s", s.name)

	s.runner.Run(containerName, containerConfig{
		Repository: "localstack/localstack",
		Tag:        "0.10.7",
		Env:        env,
		PortBindings: portBinding{
			"4575/tcp": fmt.Sprint(s.settings.SnsPort),
			"4576/tcp": fmt.Sprint(s.settings.SqsPort),
		},
		HealthCheck: localstackHealthCheck(s.runner, containerName),
		PrintLogs:   s.settings.Debug,
	})
}

func (s *snsSqsComponent) provideSnsClient() *sns.SNS {
	return s.clients.New(fmt.Sprintf("sns-%s", s.name), func() interface{} {
		sess := getAwsSession(s.settings.Host, s.settings.SnsPort)

		return sns.New(sess)
	}).(*sns.SNS)
}

func (s *snsSqsComponent) provideSqsClient() *sqs.SQS {
	return s.clients.New(fmt.Sprintf("sqs-%s", s.name), func() interface{} {
		sess := getAwsSession(s.settings.Host, s.settings.SqsPort)

		return sqs.New(sess)
	}).(*sqs.SQS)
}
