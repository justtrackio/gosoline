package test

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	componentSns    = "sns"
	componentSqs    = "sqs"
	componentSnsSqs = "sns_sqs"
)

type snsSqsSettings struct {
	*mockSettings
}

type snsSqsComponent struct {
	mockComponentBase
	settings *snsSqsSettings
	clients  *simpleCache
}

func (s *snsSqsComponent) Boot(config cfg.Config, _ log.Logger, runner *dockerRunnerLegacy, settings *mockSettings, name string) {
	s.name = name
	s.runner = runner
	s.clients = &simpleCache{}
	s.settings = &snsSqsSettings{
		mockSettings: settings,
	}
	key := fmt.Sprintf("mocks.%s", name)
	config.UnmarshalKey(key, s.settings)
}

func (s *snsSqsComponent) getContainerConfig() *containerConfigLegacy {
	services := "SERVICES=" + strings.Join([]string{
		componentSns,
		componentSqs,
	}, ",")

	env := []string{
		services,
		"EAGER_SERVICE_LOADING=1",
	}

	if s.settings.Debug {
		env = append(env, "DEBUG=1")
	}

	return &containerConfigLegacy{
		Repository: "localstack/localstack",
		Tag:        "0.13.0.4",
		Env:        env,
		PortBindings: portBindingLegacy{
			"4566/tcp": fmt.Sprint(s.settings.Port),
		},
		PortMappings: portMappingLegacy{
			"4566/tcp": &s.settings.Port,
		},
		HostMapping: hostMappingLegacy{
			dialPort: &s.settings.Port,
			setHost:  &s.settings.Host,
		},
		HealthCheck: localstackHealthCheck(s.settings.mockSettings, componentSns, componentSqs),
		PrintLogs:   s.settings.Debug,
		ExpireAfter: s.settings.ExpireAfter,
	}
}

func (s *snsSqsComponent) PullContainerImage() error {
	containerName := fmt.Sprintf("gosoline_test_sns_sqs_%s", s.name)
	containerConfig := s.getContainerConfig()

	return s.runner.PullContainerImage(containerName, containerConfig)
}

func (s *snsSqsComponent) Start() error {
	containerName := fmt.Sprintf("gosoline_test_sns_sqs_%s", s.name)
	containerConfig := s.getContainerConfig()

	return s.runner.Run(containerName, containerConfig)
}

func (s *snsSqsComponent) provideSnsClient() *sns.SNS {
	return s.clients.New(fmt.Sprintf("%s-%s", componentSns, s.name), func() interface{} {
		sess := getAwsSession(s.settings.Host, s.settings.Port)

		return sns.New(sess)
	}).(*sns.SNS)
}

func (s *snsSqsComponent) provideSqsClient() *sqs.SQS {
	return s.clients.New(fmt.Sprintf("%s-%s", componentSqs, s.name), func() interface{} {
		sess := getAwsSession(s.settings.Host, s.settings.Port)

		return sqs.New(sess)
	}).(*sqs.SQS)
}
