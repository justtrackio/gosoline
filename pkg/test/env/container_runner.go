package env

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/coffin"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/uuid"
	"github.com/cenkalti/backoff"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"net"
	"sync"
	"time"
)

type containerConfig struct {
	Repository   string
	Tag          string
	Env          []string
	Cmd          []string
	PortBindings portBindings
	ExpireAfter  time.Duration
}

type portBindings map[string]int

type containerBinding struct {
	host string
	port string
}

type container struct {
	typ      string
	name     string
	bindings map[string]containerBinding
}

type healthCheckSettings struct {
	InitialInterval time.Duration `cfg:"initial_interval" default:"1s"`
	MaxInterval     time.Duration `cfg:"max_interval" default:"3s"`
	MaxElapsedTime  time.Duration `cfg:"max_elapsed_time" default:"1m"`
}

type containerRunnerSettings struct {
	Endpoint    string              `cfg:"endpoint"`
	NamePrefix  string              `cfg:"name_prefix" default:"goso"`
	HealthCheck healthCheckSettings `cfg:"health_check"`
	ExpireAfter time.Duration       `cfg:"expire_after"`
}

type containerRunner struct {
	logger    mon.Logger
	pool      *dockertest.Pool
	id        string
	resources map[string]*dockertest.Resource
	settings  *containerRunnerSettings
}

func NewContainerRunner(config cfg.Config, logger mon.Logger) *containerRunner {
	id := uuid.New().NewV4()
	logger = logger.WithChannel("container-runner")

	settings := &containerRunnerSettings{}
	config.UnmarshalKey("test.container_runner", settings)

	pool, err := dockertest.NewPool(settings.Endpoint)

	if err != nil {
		logger.Fatalf(err, "can not create container runner")
	}

	return &containerRunner{
		logger:    logger,
		pool:      pool,
		id:        id,
		resources: make(map[string]*dockertest.Resource),
		settings:  settings,
	}
}

func (r *containerRunner) RunContainers(skeletons []*componentSkeleton) (map[string]*container, error) {
	containers := make(map[string]*container)

	if len(skeletons) == 0 {
		return containers, nil
	}

	cfn := coffin.New()
	lck := new(sync.Mutex)

	for i := range skeletons {
		if skeletons[i].containerConfig == nil {
			continue
		}

		cfn.Gof(func(skeleton *componentSkeleton) func() error {
			return func() error {
				var err error
				var container *container

				if container, err = r.RunContainer(skeleton); err != nil {
					return fmt.Errorf("can not run container %s: %w", skeleton.id(), err)
				}

				lck.Lock()
				defer lck.Unlock()

				containers[skeleton.id()] = container

				return nil
			}
		}(skeletons[i]), "can not run container %s", skeletons[i].id())
	}

	return containers, cfn.Wait()
}

func (r *containerRunner) RunContainer(skeleton *componentSkeleton) (*container, error) {
	containerName := fmt.Sprintf("%s-%s-%s", r.settings.NamePrefix, r.id, skeleton.id())
	r.logger.Infof("run container %s %s", skeleton.typ, containerName)

	config := skeleton.containerConfig
	bindings := make(map[docker.Port][]docker.PortBinding)

	for containerPort, hostPort := range config.PortBindings {
		bindings[docker.Port(containerPort)] = []docker.PortBinding{
			{
				HostPort: fmt.Sprint(hostPort),
			},
		}
	}

	runOptions := &dockertest.RunOptions{
		Name:         containerName,
		Repository:   config.Repository,
		Tag:          config.Tag,
		Env:          config.Env,
		Cmd:          config.Cmd,
		PortBindings: bindings,
	}

	resource, err := r.pool.RunWithOptions(runOptions)

	if err != nil {
		return nil, fmt.Errorf("can not run container %s: %w", skeleton.id(), err)
	}

	r.resources[skeleton.id()] = resource

	if err = r.expireAfter(resource, config.ExpireAfter); err != nil {
		return nil, fmt.Errorf("could not set expiry on container %s: %w", containerName, err)
	}

	resolvedBindings, err := r.resolveBindings(resource, config.PortBindings)

	if err != nil {
		return nil, fmt.Errorf("can not resolve bindings: %w", err)
	}

	container := &container{
		typ:      skeleton.typ,
		name:     containerName,
		bindings: resolvedBindings,
	}

	if err = r.waitUntilHealthy(container, skeleton.healthCheck); err != nil {
		return nil, fmt.Errorf("healthcheck failed on container for component %s: %w", skeleton.id(), err)
	}

	return container, err
}

func (r *containerRunner) expireAfter(resource *dockertest.Resource, expireAfter time.Duration) error {
	if r.settings.ExpireAfter > 0 {
		expireAfter = r.settings.ExpireAfter
	}

	if err := resource.Expire(uint(expireAfter.Seconds())); err != nil {
		return err
	}

	return nil
}

func (r *containerRunner) resolveBindings(resource *dockertest.Resource, bindings portBindings) (map[string]containerBinding, error) {
	var err error
	var resolvedAddresses = make(map[string]containerBinding)

	for containerPort := range bindings {
		if resolvedAddresses[containerPort], err = r.resolveBinding(resource, containerPort); err != nil {

		}
	}

	return resolvedAddresses, nil
}

func (r *containerRunner) resolveBinding(resource *dockertest.Resource, containerPort string) (containerBinding, error) {
	var err error
	var hostPort string
	var address containerBinding

	err = r.pool.Retry(func() error {
		if hostPort = resource.GetHostPort(containerPort); hostPort == "" {
			return fmt.Errorf("port is not ready yet")
		}

		return nil
	})

	if err != nil {
		return address, fmt.Errorf("can not resolve binding for port %s: %w", containerPort, err)
	}

	if address.host, address.port, err = net.SplitHostPort(hostPort); err != nil {
		return address, fmt.Errorf("could not split hostPort into host and port: %w", err)
	}

	address.host = resource.Container.NetworkSettings.Networks["bridge"].Gateway

	return address, nil
}

func (r *containerRunner) waitUntilHealthy(container *container, healthCheck ComponentHealthCheck) error {
	backoffSetting := backoff.NewExponentialBackOff()
	backoffSetting.InitialInterval = r.settings.HealthCheck.InitialInterval
	backoffSetting.MaxInterval = r.settings.HealthCheck.MaxInterval
	backoffSetting.MaxElapsedTime = r.settings.HealthCheck.MaxElapsedTime
	backoffSetting.Multiplier = 1.5
	backoffSetting.RandomizationFactor = 1

	start := time.Now()
	time.Sleep(time.Second)

	typ := container.typ
	name := container.name

	notify := func(err error, _ time.Duration) {
		r.logger.Debugf("%s %s is still unhealthy since %v: %s", typ, name, time.Since(start), err)
	}

	err := backoff.RetryNotify(func() error {
		return healthCheck(container)
	}, backoffSetting, notify)

	if err != nil {
		return err
	}

	r.logger.Infof("%s %s got healthy after %s", typ, name, time.Since(start))

	return nil
}

func (r *containerRunner) Stop() error {
	for name, resource := range r.resources {
		if err := r.pool.Purge(resource); err != nil {
			return fmt.Errorf("could not stop container %s: %w", name, err)
		}
	}

	return nil
}
