package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

type cloudwatchSettings struct {
	*mockSettings
	Port int `cfg:"port"`
}

type cloudwatchComponent struct {
	name     string
	settings *cloudwatchSettings
	clients  *simpleCache
	runner   *dockerRunner
}

func (c *cloudwatchComponent) Boot(config cfg.Config, runner *dockerRunner, settings *mockSettings, name string) {
	c.name = name
	c.runner = runner
	c.settings = &cloudwatchSettings{
		mockSettings: settings,
	}
	c.clients = &simpleCache{}
	key := fmt.Sprintf("mocks.%s", name)
	config.UnmarshalKey(key, c.settings)
}

func (c *cloudwatchComponent) Start() {
	containerName := fmt.Sprintf("gosoline_test_cloudwatch_%s", c.name)

	c.runner.Run(containerName, containerConfig{
		Repository: "localstack/localstack",
		Tag:        "0.10.7",
		Env: []string{
			"SERVICES=cloudwatch",
		},
		PortBindings: portBinding{
			"4582/tcp": fmt.Sprint(c.settings.Port),
		},
		HealthCheck: localstackHealthCheck(c.runner, containerName),
		PrintLogs:   c.settings.Debug,
	})
}

func (c *cloudwatchComponent) provideCloudwatchClient() *cloudwatch.CloudWatch {
	return c.clients.New(c.name, func() interface{} {
		sess := getAwsSession(c.settings.Host, c.settings.Port)

		return cloudwatch.New(sess)
	}).(*cloudwatch.CloudWatch)
}
