package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

const componentCloudwatch = "cloudwatch"

type cloudwatchSettings struct {
	*healthcheckMockSettings
	Port int `cfg:"port" default:"0"`
}

type cloudwatchComponent struct {
	baseComponent
	settings *cloudwatchSettings
	clients  *simpleCache
}

func (c *cloudwatchComponent) Boot(config cfg.Config, _ mon.Logger, runner *dockerRunner, settings *mockSettings, name string) {
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

func (c *cloudwatchComponent) Start() error {
	containerName := fmt.Sprintf("gosoline_test_cloudwatch_%s", c.name)

	res, err := c.runner.Run(containerName, containerConfig{
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

	if err != nil {
		return err
	}

	err = c.setPort(res, "4582/tcp", &c.settings.Port)

	if err != nil {
		return err
	}

	err = c.setPort(res, "8080/tcp", &c.settings.Healthcheck.Port)

	return err
}

func (c *cloudwatchComponent) Ports() map[string]int {
	return map[string]int{
		c.name:   c.settings.Port,
		"health": c.settings.Healthcheck.Port,
	}
}

func (c *cloudwatchComponent) provideCloudwatchClient() *cloudwatch.CloudWatch {
	return c.clients.New(c.name, func() interface{} {
		sess := getAwsSession(c.settings.Host, c.settings.Port)

		return cloudwatch.New(sess)
	}).(*cloudwatch.CloudWatch)
}
