package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"sync"
	"time"
)

type mockComponent interface {
	Boot(config cfg.Config, runner *dockerRunner, settings *mockSettings, name string)
	Start()
}

type mockSettings struct {
	Debug       bool          `cfg:"debug"`
	Component   string        `cfg:"component"`
	Host        string        `cfg:"host"`
	ExpireAfter time.Duration `cfg:"expire_after" default:"60s"`
}

type Mocks struct {
	waitGroup    sync.WaitGroup
	dockerRunner *dockerRunner
	components   map[string]mockComponent
	logger       mon.Logger
}

func newMocks() *Mocks {
	components := make(map[string]mockComponent)
	dockerRunner := newDockerRunner()
	logger := mon.NewLogger().WithChannel("mocks")

	return &Mocks{
		components:   components,
		dockerRunner: dockerRunner,
		logger:       logger,
	}
}

func (m *Mocks) Boot(config cfg.Config) {
	m.bootFromConfig(config)

	m.waitGroup.Wait()

	m.logger.Info("test environment up and running")
}

func (m *Mocks) bootFromConfig(config cfg.Config) {
	mocks := config.GetStringMap("mocks")

	for name, _ := range mocks {
		settings := &mockSettings{}
		key := fmt.Sprintf("mocks.%s", name)
		config.UnmarshalKey(key, settings)

		component := m.createComponent(settings.Component)

		m.components[name] = component
		component.Boot(config, m.dockerRunner, settings, name)
		m.runComponent(component)
	}
}

func (m *Mocks) createComponent(component string) mockComponent {
	switch component {
	case "mysql":
		return &mysqlComponent{}
	case componentSnsSqs:
		return &snsSqsComponent{}
	case componentCloudwatch:
		return &cloudwatchComponent{}
	case "dynamodb":
		return &dynamoDbComponent{}
	case "elasticsearch":
		return &elasticsearchComponent{}
	case componentKinesis:
		return &kinesisComponent{}
	case "wiremock":
		return &wiremockComponent{}
	case "redis":
		return &redisComponent{}
	default:
		panic(fmt.Errorf("unknown component type: %s", component))
	}
}

func (m *Mocks) runComponent(component mockComponent) {
	m.waitGroup.Add(1)
	go func() {
		defer m.waitGroup.Done()
		component.Start()
	}()
}

func (m *Mocks) Shutdown() {
	m.dockerRunner.RemoveAllContainers()
}

func Boot(configFilenames ...string) *Mocks {
	if len(configFilenames) == 0 {
		configFilenames = []string{"config.test.yml"}
	}

	config := cfg.New()

	for _, filename := range configFilenames {
		err := config.Option(cfg.WithConfigFile(filename, "yml"))

		if err != nil {
			panic(fmt.Errorf("failed to read config from file %s: %w", filename, err))
		}
	}

	mocks := newMocks()
	mocks.Boot(config)

	return mocks
}
