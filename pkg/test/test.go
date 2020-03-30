package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"sync"
	"time"
)

type mockComponent interface {
	Boot(config cfg.Config, logger mon.Logger, runner *dockerRunner, settings *mockSettings, name string)
	Start() error
}

type mockSettings struct {
	Debug       bool          `cfg:"debug"`
	Component   string        `cfg:"component"`
	Host        string        `cfg:"host"`
	ExpireAfter time.Duration `cfg:"expire_after" default:"60s"`
}

type mockComponentBase struct {
	logger mon.Logger
	runner *dockerRunner
	name   string
}

type Mocks struct {
	lck          sync.Mutex
	waitGroup    sync.WaitGroup
	dockerRunner *dockerRunner
	components   map[string]mockComponent
	logger       mon.Logger
	bootFailed   bool
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

func (m *Mocks) Boot(config cfg.Config) error {
	m.bootFromConfig(config)

	m.waitGroup.Wait()

	if m.bootFailed {
		return fmt.Errorf("failed to boot at least one component")
	}

	m.logger.Info("test environment up and running")

	return nil
}

func (m *Mocks) bootFromConfig(config cfg.Config) {
	mocks := config.GetStringMap("mocks")

	for name, _ := range mocks {
		settings := &mockSettings{}
		key := fmt.Sprintf("mocks.%s", name)
		config.UnmarshalKey(key, settings)

		component, err := m.createComponent(settings.Component)
		if err != nil {
			m.bootFailed = true
			continue
		}

		m.components[name] = component
		component.Boot(config, m.logger, m.dockerRunner, settings, name)

		m.runComponent(component)
	}
}

func (m *Mocks) createComponent(component string) (mockComponent, error) {
	switch component {
	case "mysql":
		return &mysqlComponent{}, nil
	case componentSnsSqs:
		return &snsSqsComponent{}, nil
	case componentCloudwatch:
		return &cloudwatchComponent{}, nil
	case "dynamodb":
		return &dynamoDbComponent{}, nil
	case "elasticsearch":
		return &elasticsearchComponent{}, nil
	case componentKinesis:
		return &kinesisComponent{}, nil
	case "wiremock":
		return &wiremockComponent{}, nil
	case "redis":
		return &redisComponent{}, nil
	default:
		return nil, fmt.Errorf("unknown component type: %s", component)
	}
}

func (m *Mocks) runComponent(component mockComponent) {
	m.waitGroup.Add(1)
	go func() {
		defer m.waitGroup.Done()

		err := component.Start()
		if err != nil {
			m.lck.Lock()
			defer m.lck.Unlock()
			m.bootFailed = true
		}
	}()
}

func (m *Mocks) Shutdown() {
	m.dockerRunner.RemoveAllContainers()
}

func Boot(configFilenames ...string) (*Mocks, error) {
	if len(configFilenames) == 0 {
		configFilenames = []string{"config.test.yml"}
	}

	config := cfg.New()

	for _, filename := range configFilenames {
		err := config.Option(cfg.WithConfigFile(filename, "yml"))

		if err != nil {
			return nil, fmt.Errorf("failed to read config from file %s: %w", filename, err)
		}
	}

	mocks := newMocks()
	err := mocks.Boot(config)

	return mocks, err
}
