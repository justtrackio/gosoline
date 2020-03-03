package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"log"
	"sync"
)

type mockComponent interface {
	Boot(name string, config cfg.Config, settings *mockSettings)
	Run(runner *dockerRunner)
	ProvideClient(clientType string) interface{}
}

type mockSettings struct {
	Debug     bool   `cfg:"debug"`
	Component string `cfg:"component"`
	Host      string `cfg:"host"`
}

type Mocks struct {
	waitGroup    sync.WaitGroup
	dockerRunner *dockerRunner
	components   map[string]mockComponent
}

func newMocks() *Mocks {
	components := make(map[string]mockComponent)
	dockerRunner := newDockerRunner()

	return &Mocks{
		components:   components,
		dockerRunner: dockerRunner,
	}
}

func (t *Mocks) Boot(config cfg.Config) {
	t.bootFromConfig(config)

	t.waitGroup.Wait()

	log.Println("test environment up and running")
	log.Println()
}

func (t *Mocks) bootFromConfig(config cfg.Config) {
	mocks := config.GetStringMap("mocks")

	for name, _ := range mocks {
		settings := &mockSettings{}
		key := fmt.Sprintf("mocks.%s", name)
		config.UnmarshalKey(key, settings)

		component := t.createComponent(settings.Component)

		t.components[name] = component
		component.Boot(name, config, settings)
		t.runComponent(component)
	}
}

func (t *Mocks) createComponent(component string) mockComponent {
	switch component {
	case "mysql":
		return &mysqlComponent{}
	case "sns_sqs":
		return &snsSqsComponent{}
	case "cloudwatch":
		return &cloudwatchComponent{}
	case "dynamodb":
		return &dynamoDbComponent{}
	case "elasticsearch":
		return &elasticsearchComponent{}
	case "kinesis":
		return &kinesisComponent{}
	case "wiremock":
		return &wiremockComponent{}
	case "redis":
		return &redisComponent{}
	default:
		panic(fmt.Errorf("unknown component type: %s", component))
	}
}

func (t *Mocks) runComponent(component mockComponent) {
	t.waitGroup.Add(1)
	go func() {
		defer t.waitGroup.Done()
		component.Run(t.dockerRunner)
	}()
}

func (t *Mocks) ProvideClient(name string, clientType string) interface{} {
	component := t.components[name]
	return component.ProvideClient(clientType)
}

func (t *Mocks) Shutdown() {
	t.dockerRunner.PurgeAllResources()
}

func Boot(configFilenames ...string) *Mocks {
	if len(configFilenames) == 0 {
		configFilenames = append(configFilenames, "config.test.yml")
	}

	config := cfg.New()

	for _, filename := range configFilenames {
		err := config.Option(cfg.WithConfigFile(filename, "yml"))

		if err != nil {
			panic(fmt.Errorf("failed to read config from file %s : %w", filename, err))
		}
	}

	mocks := newMocks()
	mocks.Boot(config)

	return mocks
}
