package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/hashicorp/go-multierror"
	"sync"
	"time"
)

type mockComponent interface {
	Boot(config cfg.Config, logger log.Logger, runner *dockerRunnerLegacy, settings *mockSettings, name string)
	Start() error
}

type mockSettings struct {
	Component   string        `cfg:"component"`
	Debug       bool          `cfg:"debug"`
	ExpireAfter time.Duration `cfg:"expire_after" default:"60s"`
	Host        string
	Tmpfs       map[string]interface{} `cfg:"tmpfs"`
}

type mockComponentBase struct {
	logger log.Logger
	runner *dockerRunnerLegacy
	name   string
}

type Mocks struct {
	waitGroup    sync.WaitGroup
	dockerRunner *dockerRunnerLegacy
	components   map[string]mockComponent
	logger       log.Logger
	errors       *multierror.Error
	errorsLock   sync.Mutex
}

func newMocks(config cfg.Config, logger log.Logger) (*Mocks, error) {
	dockerRunner := NewDockerRunnerLegacy(config)
	logger = logger.WithChannel("mocks")

	m := &Mocks{
		components:   make(map[string]mockComponent),
		dockerRunner: dockerRunner,
		logger:       logger,
	}

	m.bootFromConfig(config)

	m.waitGroup.Wait()

	err := m.errors.ErrorOrNil()
	if err != nil {
		return nil, fmt.Errorf("failed to boot at least one test component: %w", err)
	}

	m.logger.Info("test environment up and running")

	return m, nil
}

func (m *Mocks) bootFromConfig(config cfg.Config) {
	mocks := config.GetStringMap("mocks")

	for name := range mocks {
		settings := &mockSettings{}
		key := fmt.Sprintf("mocks.%s", name)
		config.UnmarshalKey(key, settings)

		component, err := m.createComponent(settings.Component)
		if err != nil {
			m.errorsLock.Lock()
			m.errors = multierror.Append(m.errors, err)
			m.errorsLock.Unlock()
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
		return &mysqlComponentLegacy{}, nil
	case componentSnsSqs:
		return &snsSqsComponent{}, nil
	case "cloudwatch":
		return &cloudwatchComponent{}, nil
	case "dynamodb":
		return &dynamoDbComponent{}, nil
	case componentKinesis:
		return &kinesisComponent{}, nil
	case componentS3:
		return &s3Component{}, nil
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
			m.errorsLock.Lock()
			m.errors = multierror.Append(m.errors, err)
			m.errorsLock.Unlock()
		}
	}()
}

func (m *Mocks) Shutdown() {
	m.dockerRunner.RemoveAllContainers()
}
