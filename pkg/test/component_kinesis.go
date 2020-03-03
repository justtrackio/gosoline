package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/aws/aws-sdk-go/service/kinesis"
)

type kinesisSettings struct {
	*mockSettings
	Port int `cfg:"port"`
}

type kinesisComponent struct {
	name     string
	settings *kinesisSettings
	clients  *simpleCache
	runner   *dockerRunner
}

func (k *kinesisComponent) Boot(config cfg.Config, runner *dockerRunner, settings *mockSettings, name string) {
	k.name = name
	k.runner = runner
	k.clients = &simpleCache{}
	k.settings = &kinesisSettings{
		mockSettings: settings,
	}
	key := fmt.Sprintf("mocks.%s", name)
	config.UnmarshalKey(key, k.settings)
}

func (k *kinesisComponent) Start() {
	containerName := fmt.Sprintf("gosoline_test_kinesis_%s", k.name)

	k.runner.Run(containerName, containerConfig{
		Repository: "localstack/localstack",
		Tag:        "0.10.7",
		Env: []string{
			"SERVICES=kinesis",
		},
		PortBindings: portBinding{
			"4568/tcp": fmt.Sprint(k.settings.Port),
		},
		HealthCheck: localstackHealthCheck(k.runner, containerName),
		PrintLogs:   k.settings.Debug,
	})
}

func (k *kinesisComponent) provideKinesisClient() *kinesis.Kinesis {
	return k.clients.New(k.name, func() interface{} {
		sess := getAwsSession(k.settings.Host, k.settings.Port)

		return kinesis.New(sess)
	}).(*kinesis.Kinesis)
}
