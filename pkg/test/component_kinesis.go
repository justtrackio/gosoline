package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/aws/aws-sdk-go/service/kinesis"
)

const componentKinesis = "kinesis"

type kinesisSettings struct {
	*healthcheckMockSettings
	Port int `cfg:"port" default:"0"`
}

type kinesisComponent struct {
	baseComponent
	settings *kinesisSettings
	clients  *simpleCache
}

func (k *kinesisComponent) Boot(config cfg.Config, _ mon.Logger, runner *dockerRunner, settings *mockSettings, name string) {
	k.name = name
	k.runner = runner
	k.clients = &simpleCache{}
	k.settings = &kinesisSettings{
		healthcheckMockSettings: &healthcheckMockSettings{
			mockSettings: settings,
			Healthcheck:  healthcheckSettings(config, name),
		},
	}
	key := fmt.Sprintf("mocks.%s", name)
	config.UnmarshalKey(key, k.settings)
}

func (k *kinesisComponent) Start() error {
	containerName := fmt.Sprintf("gosoline_test_kinesis_%s", k.name)

	res, err := k.runner.Run(containerName, containerConfig{
		Repository: "localstack/localstack",
		Tag:        "0.10.8",
		Env: []string{
			fmt.Sprintf("SERVICES=%s", componentKinesis),
		},
		PortBindings: portBinding{
			"4568/tcp": fmt.Sprint(k.settings.Port),
			"8080/tcp": fmt.Sprint(k.settings.Healthcheck.Port),
		},
		HealthCheck: localstackHealthCheck(k.settings.healthcheckMockSettings, componentKinesis),
		PrintLogs:   k.settings.Debug,
		ExpireAfter: k.settings.ExpireAfter,
	})

	if err != nil {
		return err
	}

	err = k.setPort(res, "4568/tcp", &k.settings.Port)

	if err != nil {
		return err
	}

	err = k.setPort(res, "8080/tcp", &k.settings.Healthcheck.Port)

	return err
}

func (k *kinesisComponent) Ports() map[string]int {
	return map[string]int{
		k.name:   k.settings.Port,
		"health": k.settings.Healthcheck.Port,
	}
}

func (k *kinesisComponent) provideKinesisClient() *kinesis.Kinesis {
	return k.clients.New(k.name, func() interface{} {
		sess := getAwsSession(k.settings.Host, k.settings.Port)

		return kinesis.New(sess)
	}).(*kinesis.Kinesis)
}
