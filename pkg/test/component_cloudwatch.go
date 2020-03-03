package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"log"
)

type cloudwatchSettings struct {
	*mockSettings
	Port int `cfg:"port"`
}

type cloudwatchComponent struct {
	name     string
	settings *cloudwatchSettings
	clients  *simpleCache
}

func (m *cloudwatchComponent) Boot(name string, config cfg.Config, settings *mockSettings) {
	m.name = name
	m.settings = &cloudwatchSettings{
		mockSettings: settings,
	}
	m.clients = &simpleCache{}
	key := fmt.Sprintf("mocks.%s", name)
	config.UnmarshalKey(key, m.settings)
}

func (m *cloudwatchComponent) Run(runner *dockerRunner) {
	defer log.Printf("%s component of type %s is ready", m.name, "cloudwatch")

	containerName := fmt.Sprintf("gosoline_test_cloudwatch_%s", m.name)

	runner.Run(containerName, containerConfig{
		Repository: "localstack/localstack",
		Tag:        "0.10.7",
		Env: []string{
			"SERVICES=cloudwatch",
		},
		PortBindings: portBinding{
			"4582/tcp": fmt.Sprint(m.settings.Port),
		},
		HealthCheck: localstackHealthCheck(runner, containerName),
		PrintLogs:   m.settings.Debug,
	})
}

func (m *cloudwatchComponent) ProvideClient(string) interface{} {
	return m.provideCloudwatchClient()
}

func (m *cloudwatchComponent) provideCloudwatchClient() *cloudwatch.CloudWatch {
	return m.clients.New("cloudwatch", func() interface{} {
		sess, err := getAwsSession(m.settings.Host, m.settings.Port)

		if err != nil {
			panic(fmt.Errorf("could not create cloudwatch client for %s: %w", m.name, err))
		}

		return cloudwatch.New(sess)
	}).(*cloudwatch.CloudWatch)
}
