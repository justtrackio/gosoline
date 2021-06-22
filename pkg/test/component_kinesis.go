package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/aws/aws-sdk-go/service/kinesis"
)

const componentKinesis = "kinesis"

type kinesisSettings struct {
	*healthCheckMockSettings
	Port int `cfg:"port" default:"0"`
}

type kinesisComponent struct {
	mockComponentBase
	settings *kinesisSettings
	clients  *simpleCache
}

func (k *kinesisComponent) Boot(config cfg.Config, _ log.Logger, runner *dockerRunnerLegacy, settings *mockSettings, name string) {
	k.name = name
	k.runner = runner
	k.clients = &simpleCache{}
	k.settings = &kinesisSettings{
		healthCheckMockSettings: &healthCheckMockSettings{
			mockSettings: settings,
			Healthcheck:  healthCheckSettings(config, name),
		},
	}
	key := fmt.Sprintf("mocks.%s", name)
	config.UnmarshalKey(key, k.settings)
}

func (k *kinesisComponent) Start() error {
	containerName := fmt.Sprintf("gosoline_test_kinesis_%s", k.name)

	return k.runner.Run(containerName, &containerConfigLegacy{
		Repository: "localstack/localstack",
		Tag:        "0.10.8",
		Env: []string{
			fmt.Sprintf("SERVICES=%s", componentKinesis),
		},
		PortBindings: portBindingLegacy{
			"4568/tcp": fmt.Sprint(k.settings.Port),
			"8080/tcp": fmt.Sprint(k.settings.Healthcheck.Port),
		},
		PortMappings: portMappingLegacy{
			"4568/tcp": &k.settings.Port,
			"8080/tcp": &k.settings.Healthcheck.Port,
		},
		HostMapping: hostMappingLegacy{
			dialPort: &k.settings.Port,
			setHost:  &k.settings.Host,
		},
		HealthCheck: localstackHealthCheck(k.settings.healthCheckMockSettings, componentKinesis),
		PrintLogs:   k.settings.Debug,
		ExpireAfter: k.settings.ExpireAfter,
	})
}

func (k *kinesisComponent) provideKinesisClient() *kinesis.Kinesis {
	return k.clients.New(k.name, func() interface{} {
		sess := getAwsSession(k.settings.Host, k.settings.Port)

		return kinesis.New(sess)
	}).(*kinesis.Kinesis)
}
