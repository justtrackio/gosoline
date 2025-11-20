package env

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/log"
)

type ContainerManager interface {
	RunContainers(ctx context.Context, skeletons []*componentSkeleton) error
	Stop(ctx context.Context) error
}

var _ ContainerManager = (*containerManager)(nil)

type ContainerManagerSettings struct {
	RunnerType  string              `cfg:"runner_type"  default:"local"`
	NamePrefix  string              `cfg:"name_prefix"  default:"goso"`
	HealthCheck HealthCheckSettings `cfg:"health_check"`
	ExpireAfter time.Duration       `cfg:"expire_after" default:"5m"`
}

type HealthCheckSettings struct {
	InitialInterval time.Duration `cfg:"initial_interval" default:"1s"`
	MaxInterval     time.Duration `cfg:"max_interval"     default:"3s"`
	MaxElapsedTime  time.Duration `cfg:"max_elapsed_time" default:"1m"`
}

type containerManager struct {
	logger            log.Logger
	runnerFactory     func(typ string) (ContainerRunner, error)
	runners           map[string]ContainerRunner
	settings          *ContainerManagerSettings
	shutdownCallbacks map[string]func() error
	testName          string
}

func NewContainerManager(config cfg.Config, logger log.Logger, testName string) (ContainerManager, error) {
	var err error
	runners := map[string]ContainerRunner{}

	settings := &ContainerManagerSettings{}
	if err = config.UnmarshalKey("test.container_manager", settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal container manager settings: %w", err)
	}

	runnerFactory := func(typ string) (ContainerRunner, error) {
		var err error
		var runner ContainerRunner

		if _, ok := containerRunnerFactories[typ]; !ok {
			return nil, fmt.Errorf("there is no container runner factory for type %s", typ)
		}

		if runner, err = containerRunnerFactories[typ](config, logger, settings); err != nil {
			return nil, fmt.Errorf("can not create container runner for type %s: %w", typ, err)
		}

		return runner, nil
	}

	if runners[settings.RunnerType], err = runnerFactory(settings.RunnerType); err != nil {
		return nil, err
	}

	return &containerManager{
		logger:            logger,
		runnerFactory:     runnerFactory,
		runners:           runners,
		settings:          settings,
		shutdownCallbacks: make(map[string]func() error),
		testName:          testName,
	}, nil
}

func (m *containerManager) RunContainers(ctx context.Context, skeletons []*componentSkeleton) error {
	if len(skeletons) == 0 {
		return nil
	}

	cfn := coffin.New()
	lck := new(sync.Mutex)

	for i := range skeletons {
		for containerName, description := range skeletons[i].containerDescriptions {
			skeleton := skeletons[i]

			cfn.Gof(func() error {
				var err error
				var container *Container

				request := ContainerRequest{
					TestName:             m.testName,
					ComponentType:        skeleton.typ,
					ComponentName:        skeleton.name,
					ContainerName:        containerName,
					ContainerDescription: description,
					ExpireAfter:          m.settings.ExpireAfter,
				}

				if container, err = m.runContainer(ctx, request); err != nil {
					return fmt.Errorf("could not run container %s: %w", containerName, err)
				}

				lck.Lock()
				defer lck.Unlock()

				skeleton.containers[containerName] = container

				if description.ShutdownCallback != nil {
					m.shutdownCallbacks[request.id()] = description.ShutdownCallback(container)
				}

				return nil
			}, "can not run container %s", skeleton.id())
		}
	}

	if err := cfn.Wait(); err != nil {
		return fmt.Errorf("could not run all containers: %w", err)
	}

	return nil
}

func (m *containerManager) runContainer(ctx context.Context, request ContainerRequest) (*Container, error) {
	var err error
	var container *Container

	desc := request.ContainerDescription
	runnerType := desc.ContainerConfig.RunnerType

	if runnerType == "" {
		runnerType = m.settings.RunnerType
	}

	if _, ok := m.runners[runnerType]; !ok {
		if m.runners[runnerType], err = m.runnerFactory(runnerType); err != nil {
			return nil, fmt.Errorf("can not create container runner for type %s: %w", runnerType, err)
		}
	}

	if container, err = m.runners[runnerType].RunContainer(ctx, request); err != nil {
		return nil, fmt.Errorf(
			"can not run container %s (%s:%s): %w",
			request.id(),
			desc.ContainerConfig.Repository,
			desc.ContainerConfig.Tag,
			err,
		)
	}

	if err = m.waitUntilHealthy(ctx, container, desc.HealthCheck); err != nil {
		return nil, fmt.Errorf("healthcheck failed on container for component %s: %w", request.id(), err)
	}

	return container, nil
}

func (m *containerManager) Stop(ctx context.Context) error {
	for name, cb := range m.shutdownCallbacks {
		err := cb()
		if err != nil {
			m.logger.Error(ctx, "shutdown callback failed for container %s: %w", name, err)
		}
	}

	for name, runner := range m.runners {
		if err := runner.Stop(ctx); err != nil {
			return fmt.Errorf("stopping runner %s failed: %w", name, err)
		}
	}

	return nil
}

func (m *containerManager) waitUntilHealthy(ctx context.Context, container *Container, healthCheck ComponentHealthCheck) error {
	backoffSetting := backoff.NewExponentialBackOff()
	backoffSetting.InitialInterval = m.settings.HealthCheck.InitialInterval
	backoffSetting.MaxInterval = m.settings.HealthCheck.MaxInterval
	backoffSetting.MaxElapsedTime = m.settings.HealthCheck.MaxElapsedTime
	backoffSetting.Multiplier = 1.5
	backoffSetting.RandomizationFactor = 1

	start := time.Now()
	time.Sleep(time.Second)

	typ := container.typ
	name := container.name

	notify := func(err error, _ time.Duration) {
		m.logger.Debug(ctx, "%s %s is still unhealthy since %v: %s", typ, name, time.Since(start), err)
	}

	err := backoff.RetryNotify(func() error {
		return healthCheck(container)
	}, backoffSetting, notify)
	if err != nil {
		return err
	}

	m.logger.Debug(ctx, "%s %s got healthy after %s", typ, name, time.Since(start))

	return nil
}
