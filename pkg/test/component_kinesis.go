package test

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

const componentKinesis = "kinesis"

type kinesisSettings struct {
	*mockSettings
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
		mockSettings: settings,
	}
	key := fmt.Sprintf("mocks.%s", name)
	config.UnmarshalKey(key, k.settings)
}

func (k *kinesisComponent) Start() error {
	containerName := fmt.Sprintf("gosoline_test_kinesis_%s", k.name)

	return k.runner.Run(containerName, &containerConfigLegacy{
		Repository: "localstack/localstack",
		Tag:        "0.13.0.4",
		Env: []string{
			fmt.Sprintf("SERVICES=%s", componentKinesis),
			"EAGER_SERVICE_LOADING=1",
		},
		PortBindings: portBindingLegacy{
			"4566/tcp": fmt.Sprint(k.settings.Port),
		},
		PortMappings: portMappingLegacy{
			"4566/tcp": &k.settings.Port,
		},
		HostMapping: hostMappingLegacy{
			dialPort: &k.settings.Port,
			setHost:  &k.settings.Host,
		},
		HealthCheck: localstackHealthCheck(k.settings.mockSettings, componentKinesis),
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
