package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/aws/aws-sdk-go/service/s3"
)

const componentS3 = "s3"

type s3Settings struct {
	*healthCheckMockSettings
	Port int `cfg:"port" default:"0"`
}

type s3Component struct {
	mockComponentBase
	settings *s3Settings
	clients  *simpleCache
}

func (k *s3Component) Boot(config cfg.Config, _ mon.Logger, runner *dockerRunnerLegacy, settings *mockSettings, name string) {
	k.name = name
	k.runner = runner
	k.clients = &simpleCache{}
	k.settings = &s3Settings{
		healthCheckMockSettings: &healthCheckMockSettings{
			mockSettings: settings,
			Healthcheck:  healthCheckSettings(config, name),
		},
	}
	key := fmt.Sprintf("mocks.%s", name)
	config.UnmarshalKey(key, k.settings)
}

func (k *s3Component) Start() error {
	containerName := fmt.Sprintf("gosoline_test_s3_%s", k.name)

	return k.runner.Run(containerName, &containerConfigLegacy{
		Repository: "localstack/localstack",
		Tag:        "0.10.8",
		Env: []string{
			fmt.Sprintf("SERVICES=%s", componentS3),
		},
		PortBindings: portBindingLegacy{
			"4572/tcp": fmt.Sprint(k.settings.Port),
			"8080/tcp": fmt.Sprint(k.settings.Healthcheck.Port),
		},
		PortMappings: portMappingLegacy{
			"4572/tcp": &k.settings.Port,
			"8080/tcp": &k.settings.Healthcheck.Port,
		},
		HostMapping: hostMappingLegacy{
			dialPort: &k.settings.Port,
			setHost:  &k.settings.Host,
		},
		HealthCheck: localstackHealthCheck(k.settings.healthCheckMockSettings, componentS3),
		PrintLogs:   k.settings.Debug,
		ExpireAfter: k.settings.ExpireAfter,
	})
}

func (k *s3Component) provideS3Client() *s3.S3 {
	return k.clients.New(k.name, func() interface{} {
		sess := getAwsSession(k.settings.Host, k.settings.Port)

		return s3.New(sess)
	}).(*s3.S3)
}
