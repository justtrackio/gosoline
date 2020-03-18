package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

const componentCloudwatch = "cloudwatch"

type cloudwatchSettings struct {
	*healthcheckMockSettings
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
	c.clients = &simpleCache{}
	c.settings = &cloudwatchSettings{
		healthcheckMockSettings: &healthcheckMockSettings{
			mockSettings: settings,
			Healthcheck:  healthcheckSettings(config, name),
		},
	}
	key := fmt.Sprintf("mocks.%s", name)
	config.UnmarshalKey(key, c.settings)
}

func (c *cloudwatchComponent) Start() {
	containerName := fmt.Sprintf("gosoline_test_cloudwatch_%s", c.name)

	c.runner.Run(containerName, containerConfig{
		Repository: "localstack/localstack",
		Tag:        "0.10.8",
		Env: []string{
			"SERVICES=cloudwatch",
		},
		PortBindings: portBinding{
			"4582/tcp": fmt.Sprint(c.settings.Port),
			"8080/tcp": fmt.Sprint(c.settings.Healthcheck.Port),
		},
		HealthCheck: localstackHealthCheck(c.settings.healthcheckMockSettings, componentCloudwatch),
		PrintLogs:   c.settings.Debug,
		ExpireAfter: c.settings.ExpireAfter,
	})
}

func (c *cloudwatchComponent) provideCloudwatchClient() *cloudwatch.CloudWatch {
	return c.clients.New(c.name, func() interface{} {
		sess := getAwsSession(c.settings.Host, c.settings.Port)

		return cloudwatch.New(sess)
	}).(*cloudwatch.CloudWatch)
}
