package env

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/coffin"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/uuid"
	"github.com/cenkalti/backoff"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"
)

type containerConfig struct {
	Repository   string
	Tmpfs        []TmpfsSettings
	Tag          string
	Env          []string
	Cmd          []string
	PortBindings portBindings
	ExposedPorts []string
	ExpireAfter  time.Duration
}

type portBindings map[string]int

type containerBinding struct {
	host string
	port string
}

func (b containerBinding) getAddress() string {
	return fmt.Sprintf("%s:%s", b.host, b.port)
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

type authSettings struct {
	Username      string `cfg:"username" default:""`
	Password      string `cfg:"password" default:""`
	Email         string `cfg:"email" default:""`
	ServerAddress string `cfg:"server_address" default:""`
}

func (a authSettings) GetAuthConfig() docker.AuthConfiguration {
	return docker.AuthConfiguration{
		Username:      a.Username,
		Password:      a.Password,
		Email:         a.Email,
		ServerAddress: a.ServerAddress,
	}
}

type containerRunnerSettings struct {
	Endpoint    string              `cfg:"endpoint"`
	NamePrefix  string              `cfg:"name_prefix" default:"goso"`
	HealthCheck healthCheckSettings `cfg:"health_check"`
	ExpireAfter time.Duration       `cfg:"expire_after"`
	Auth        authSettings        `cfg:"auth"`
}

type containerRunner struct {
	logger       log.Logger
	pool         *dockertest.Pool
	id           string
	resources    map[string]*dockertest.Resource
	resourcesLck sync.Mutex
	settings     *containerRunnerSettings
}

func NewContainerRunner(config cfg.Config, logger log.Logger) (*containerRunner, error) {
	id := uuid.New().NewV4()
	logger = logger.WithChannel("container-runner")

	settings := &containerRunnerSettings{}
	config.UnmarshalKey("test.container_runner", settings)

	pool, err := dockertest.NewPool(settings.Endpoint)

	if err != nil {
		return nil, fmt.Errorf("can not create docker pool: %w", err)
	}

	return &containerRunner{
		logger:       logger,
		pool:         pool,
		id:           id,
		resources:    make(map[string]*dockertest.Resource),
		resourcesLck: sync.Mutex{},
		settings:     settings,
	}, nil
}

func (r *containerRunner) RunContainers(skeletons []*componentSkeleton) error {
	if len(skeletons) == 0 {
		return nil
	}

	cfn := coffin.New()
	lck := new(sync.Mutex)

	for i := range skeletons {
		for name, description := range skeletons[i].containerDescriptions {
			name := name
			description := description
			skeleton := skeletons[i]

			cfn.Gof(func() error {
				var err error
				var container *container

				if container, err = r.RunContainer(skeleton, name, description); err != nil {
					return fmt.Errorf("can not run container %s: %w", skeleton.id(), err)
				}

				lck.Lock()
				defer lck.Unlock()

				skeleton.containers[name] = container

				return nil
			}, "can not run container %s", skeleton.id())
		}
	}

	return cfn.Wait()
}

func (r *containerRunner) RunContainer(skeleton *componentSkeleton, name string, description *componentContainerDescription) (*container, error) {
	containerName := fmt.Sprintf("%s-%s-%s-%s", r.settings.NamePrefix, r.id, skeleton.id(), name)
	r.logger.Debug("run container %s %s", skeleton.typ, containerName)

	config := description.containerConfig
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
		ExposedPorts: config.ExposedPorts,
		Auth:         r.settings.Auth.GetAuthConfig(),
	}

	tmpfsConfig := r.getTmpfsConfig(config.Tmpfs)

	resource, err := r.pool.RunWithOptions(runOptions, func(hc *docker.HostConfig) {
		hc.AutoRemove = true
		hc.Tmpfs = tmpfsConfig
	})

	if err != nil {
		return nil, fmt.Errorf("can not run container %s: %w", skeleton.id(), err)
	}

	resourceId := fmt.Sprintf("%s-%s", skeleton.id(), name)
	r.resourcesLck.Lock()
	r.resources[resourceId] = resource
	r.resourcesLck.Unlock()

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

	if err = r.waitUntilHealthy(container, description.healthCheck); err != nil {
		return nil, fmt.Errorf("healthcheck failed on container for component %s: %w", skeleton.id(), err)
	}

	return container, err
}

func (r *containerRunner) getTmpfsConfig(settings []TmpfsSettings) map[string]string {
	config := make(map[string]string)

	for _, setting := range settings {
		params := make([]string, 0)

		if setting.Size != "" {
			params = append(params, fmt.Sprintf("size=%s", setting.Size))
		}

		if setting.Mode != "" {
			params = append(params, fmt.Sprintf("mode=%s", setting.Mode))
		}

		config[setting.Path] = strings.Join(params, ",")
	}

	return config
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
			return nil, fmt.Errorf("failed to resolve binding: %w", err)
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
		r.logger.Debug("%s %s is still unhealthy since %v: %s", typ, name, time.Since(start), err)
	}

	err := backoff.RetryNotify(func() error {
		return healthCheck(container)
	}, backoffSetting, notify)

	if err != nil {
		return err
	}

	r.logger.Debug("%s %s got healthy after %s", typ, name, time.Since(start))

	return nil
}

var alreadyExists = regexp.MustCompile(`API error \(409\): removal of container (\w+) is already in progress`)

func (r *containerRunner) Stop() error {
	for name, resource := range r.resources {
		if err := r.pool.Purge(resource); err != nil {
			if !alreadyExists.MatchString(err.Error()) {
				return fmt.Errorf("could not stop container %s: %w", name, err)
			}

			r.logger.Debug("someone else is already stopping container %s, ignoring error %s", name, err.Error())
		}

		r.logger.Debug("stopping container %s", name)
	}

	return nil
}
