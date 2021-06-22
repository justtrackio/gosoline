package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

type cloudwatchSettingsLegacy struct {
	*healthCheckMockSettings
	Port int `cfg:"port" default:"0"`
}

type cloudwatchComponent struct {
	mockComponentBase
	settings *cloudwatchSettingsLegacy
	clients  *simpleCache
}

func (c *cloudwatchComponent) Boot(config cfg.Config, _ log.Logger, runner *dockerRunnerLegacy, settings *mockSettings, name string) {
	c.name = name
	c.runner = runner
	c.clients = &simpleCache{}
	c.settings = &cloudwatchSettingsLegacy{
		healthCheckMockSettings: &healthCheckMockSettings{
			mockSettings: settings,
			Healthcheck:  healthCheckSettings(config, name),
		},
	}
	key := fmt.Sprintf("mocks.%s", name)
	config.UnmarshalKey(key, c.settings)
}

func (c *cloudwatchComponent) Start() error {
	containerName := fmt.Sprintf("gosoline_test_cloudwatch_%s", c.name)

	return c.runner.Run(containerName, &containerConfigLegacy{
		Repository: "localstack/localstack",
		Tag:        "0.10.8",
		Env: []string{
			"SERVICES=cloudwatch",
		},
		PortBindings: portBindingLegacy{
			"4582/tcp": fmt.Sprint(c.settings.Port),
			"8080/tcp": fmt.Sprint(c.settings.Healthcheck.Port),
		},
		PortMappings: portMappingLegacy{
			"4582/tcp": &c.settings.Port,
			"8080/tcp": &c.settings.Healthcheck.Port,
		},
		HostMapping: hostMappingLegacy{
			dialPort: &c.settings.Port,
			setHost:  &c.settings.Host,
		},
		HealthCheck: localstackHealthCheck(c.settings.healthCheckMockSettings, "cloudwatch"),
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
