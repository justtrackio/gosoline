package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"log"
)

type kinesisSettings struct {
	*mockSettings
	Port int `cfg:"port"`
}

type kinesisComponent struct {
	name     string
	settings *kinesisSettings
	clients  *simpleCache
}

func (m *kinesisComponent) Boot(name string, config cfg.Config, settings *mockSettings) {
	m.name = name
	m.clients = &simpleCache{}
	m.settings = &kinesisSettings{
		mockSettings: settings,
	}
	key := fmt.Sprintf("mocks.%s", name)
	config.UnmarshalKey(key, m.settings)
}

func (m *kinesisComponent) Run(runner *dockerRunner) {
	defer log.Printf("%s component of type %s is ready", m.name, "kinesis")

	containerName := fmt.Sprintf("gosoline_test_kinesis_%s", m.name)

	runner.Run(containerName, containerConfig{
		Repository: "localstack/localstack",
		Tag:        "0.10.7",
		Env: []string{
			"SERVICES=kinesis",
		},
		PortBindings: portBinding{
			"4568/tcp": fmt.Sprint(m.settings.Port),
		},
		HealthCheck: localstackHealthCheck(runner, containerName),
		PrintLogs:   m.settings.Debug,
	})
}

func (m *kinesisComponent) ProvideClient(string) interface{} {
	return m.provideKinesisClient()
}

func (m *kinesisComponent) provideKinesisClient() *kinesis.Kinesis {
	return m.clients.New("kinesis", func() interface{} {

		sess, err := getAwsSession(m.settings.Host, m.settings.Port)

		if err != nil {
			panic(fmt.Errorf("could not create kinesis client %s: %w", m.name, err))
		}

		return kinesis.New(sess)

	}).(*kinesis.Kinesis)
}
