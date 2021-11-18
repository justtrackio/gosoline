package test

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type cloudwatchSettingsLegacy struct {
	*mockSettings
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
		mockSettings: settings,
	}
	key := fmt.Sprintf("mocks.%s", name)
	config.UnmarshalKey(key, c.settings)
}

func (c *cloudwatchComponent) Start() error {
	containerName := fmt.Sprintf("gosoline_test_cloudwatch_%s", c.name)

	return c.runner.Run(containerName, &containerConfigLegacy{
		Repository: "localstack/localstack",
		Tag:        "0.12.5",
		Env: []string{
			"SERVICES=cloudwatch",
			"EAGER_SERVICE_LOADING=1",
		},
		PortBindings: portBindingLegacy{
			"4566/tcp": fmt.Sprint(c.settings.Port),
		},
		PortMappings: portMappingLegacy{
			"4566/tcp": &c.settings.Port,
		},
		HostMapping: hostMappingLegacy{
			dialPort: &c.settings.Port,
			setHost:  &c.settings.Host,
		},
		HealthCheck: localstackHealthCheck(c.settings.mockSettings, "cloudwatch"),
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
