package env

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/uuid"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

type containerConfig struct {
	Auth                 authSettings
	Repository           string
	Tmpfs                []TmpfsSettings
	Tag                  string
	Env                  []string
	Cmd                  []string
	PortBindings         portBindings
	ExposedPorts         []string
	ExpireAfter          time.Duration
	UseExternalContainer bool
	ContainerBindings    containerBindings
}

type portBindings map[string]int

type containerBindings map[string]containerBinding

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

// IsEmpty returns true when username and password are empty as both are the minimum needed for authentication
func (a authSettings) IsEmpty() bool {
	return a.Username == "" && a.Password == ""
}

type containerRunnerSettings struct {
	Endpoint    string              `cfg:"endpoint"`
	NamePrefix  string              `cfg:"name_prefix" default:"goso"`
	HealthCheck healthCheckSettings `cfg:"health_check"`
	ExpireAfter time.Duration       `cfg:"expire_after"`
	Auth        authSettings        `cfg:"auth"`
}

type containerRunner struct {
	logger            log.Logger
	pool              *dockertest.Pool
	id                string
	resources         map[string]*dockertest.Resource
	resourcesLck      sync.Mutex
	settings          *containerRunnerSettings
	shutdownCallbacks map[string]func() error
}

func NewContainerRunner(config cfg.Config, logger log.Logger) (*containerRunner, error) {
	id := uuid.New().NewV4()
	logger = logger.WithChannel("container-runner")

	settings := &containerRunnerSettings{}
	if err := config.UnmarshalKey("test.container_runner", settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal container runner settings: %w", err)
	}

	pool, err := dockertest.NewPool(settings.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("can not create docker pool: %w", err)
	}

	// do this here, if we let the call to `pool.Retry` do this, we trigger a data race (not bad in this case, but annoying
	// for the race detector)
	if pool.MaxWait == 0 {
		pool.MaxWait = time.Minute
	}

	return &containerRunner{
		logger:            logger,
		pool:              pool,
		id:                id,
		resources:         make(map[string]*dockertest.Resource),
		resourcesLck:      sync.Mutex{},
		settings:          settings,
		shutdownCallbacks: make(map[string]func() error, 0),
	}, nil
}

func (r *containerRunner) PullContainerImages(skeletons []*componentSkeleton) error {
	cfn := coffin.New()
	cfn.Go(func() error {
		for _, skeleton := range skeletons {
			for _, description := range skeleton.containerDescriptions {
				if description.containerConfig.UseExternalContainer {
					continue
				}

				cfn.Gof(func() error {
					return r.PullContainerImage(description)
				}, "can not pull container %s", skeleton.id())
			}
		}

		return nil
	})

	return cfn.Wait()
}

func (r *containerRunner) PullContainerImage(description *componentContainerDescription) error {
	config := description.containerConfig
	imageName := fmt.Sprintf("%s:%s", config.Repository, config.Tag)
	_, err := r.pool.Client.InspectImage(imageName)
	if err != nil && !errors.Is(err, docker.ErrNoSuchImage) {
		return fmt.Errorf("could not check if image %s exists: %w", imageName, err)
	}

	if err == nil {
		return nil
	}

	pullImageOptions := docker.PullImageOptions{
		Repository: config.Repository,
		Tag:        config.Tag,
	}

	containerAuth := description.containerConfig.Auth
	authConfig := r.getAuthConfig(containerAuth)

	err = r.pool.Client.PullImage(pullImageOptions, authConfig)
	if err != nil {
		return fmt.Errorf("could not pull image %q: %w", imageName, err)
	}

	return nil
}

func (r *containerRunner) getAuthConfig(containerAuth authSettings) docker.AuthConfiguration {
	if !containerAuth.IsEmpty() {
		return containerAuth.GetAuthConfig()
	}

	return r.settings.Auth.GetAuthConfig()
}

func (r *containerRunner) RunContainers(skeletons []*componentSkeleton) error {
	if len(skeletons) == 0 {
		return nil
	}

	if err := r.PullContainerImages(skeletons); err != nil {
		return err
	}

	cfn := coffin.New()
	lck := new(sync.Mutex)

	for i := range skeletons {
		for name, description := range skeletons[i].containerDescriptions {
			skeleton := skeletons[i]

			cfn.Gof(func() error {
				var err error
				var container *container

				if container, err = r.RunContainer(skeleton, name, description); err != nil {
					return fmt.Errorf("can not run container %s (%s:%s): %w", skeleton.id(), description.containerConfig.Repository, description.containerConfig.Tag, err)
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
	config := description.containerConfig
	containerName := fmt.Sprintf("%s-%s-%s-%s", r.settings.NamePrefix, r.id, skeleton.id(), name)

	var container *container
	var err error

	if config.UseExternalContainer {
		container, err = r.createExternalContainer(containerName, skeleton.typ, description.containerConfig)
		if err != nil {
			return nil, fmt.Errorf("could not create external container: %w", err)
		}
	} else {
		container, err = r.runNewContainer(containerName, skeleton, name, config)
		if err != nil {
			return nil, fmt.Errorf("could not run new container: %w", err)
		}
	}

	if err = r.waitUntilHealthy(container, description.healthCheck); err != nil {
		if description.containerConfig.UseExternalContainer {
			return nil, fmt.Errorf("healthcheck failed on container for component %s. The container is configured to use an external container, is that container running? %w", skeleton.id(), err)
		}

		return nil, fmt.Errorf("healthcheck failed on container for component %s: %w", skeleton.id(), err)
	}

	if description.shutdownCallback != nil {
		if _, exists := r.shutdownCallbacks[name]; exists {
			return nil, fmt.Errorf("there already exists a shutdown callback for %s", name)
		}

		r.shutdownCallbacks[name] = description.shutdownCallback(container)
	}

	return container, err
}

func (r *containerRunner) createExternalContainer(containerName, skeletonTyp string, config *containerConfig) (*container, error) {
	r.logger.Debug("create external container %s %s", skeletonTyp, containerName)

	containerBindings := make(map[string]containerBinding, len(config.ContainerBindings))
	for key, cb := range config.ContainerBindings {
		containerBindings[key] = containerBinding{
			host: cb.host,
			port: cb.port,
		}
	}

	return &container{
		typ:      skeletonTyp,
		name:     containerName,
		bindings: containerBindings,
	}, nil
}

func (r *containerRunner) runNewContainer(containerName string, skeleton *componentSkeleton, name string, config *containerConfig) (*container, error) {
	r.logger.Debug("run container %s %s:%s %s", skeleton.typ, config.Repository, config.Tag, containerName)

	bindings := make(map[docker.Port][]docker.PortBinding)

	for containerPort, hostPort := range config.PortBindings {
		bindings[docker.Port(containerPort)] = []docker.PortBinding{
			{
				HostPort: fmt.Sprint(hostPort),
			},
		}
	}

	containerAuth := config.Auth
	authConfig := r.getAuthConfig(containerAuth)

	runOptions := &dockertest.RunOptions{
		Name:         containerName,
		Repository:   config.Repository,
		Tag:          config.Tag,
		Env:          config.Env,
		Cmd:          config.Cmd,
		PortBindings: bindings,
		ExposedPorts: config.ExposedPorts,
		Auth:         authConfig,
	}

	tmpfsConfig := r.getTmpfsConfig(config.Tmpfs)

	resourceId := fmt.Sprintf("%s-%s", skeleton.id(), name)

	container, err := r.runContainer(runOptions, resourceId, config.ExpireAfter, config.PortBindings, skeleton.typ, tmpfsConfig)
	if err != nil {
		return nil, err
	}

	return container, nil
}

func (r *containerRunner) runContainer(options *dockertest.RunOptions, resourceId string, expireAfter time.Duration, bindings portBindings, typ string, tmpfs map[string]string) (*container, error) {
	var resourceContainer *container

	resource, err := r.pool.RunWithOptions(options, func(hc *docker.HostConfig) {
		hc.AutoRemove = true
		hc.Tmpfs = tmpfs
	})
	if err != nil {
		return nil, fmt.Errorf("can not run container %s: %w", resourceId, err)
	}

	r.resourcesLck.Lock()
	r.resources[resourceId] = resource
	r.resourcesLck.Unlock()

	if err = r.expireAfter(resource, expireAfter); err != nil {
		return nil, fmt.Errorf("could not set expiry on container %s: %w", options.Name, err)
	}

	resolvedBindings, err := r.resolveBindings(resource, bindings)
	if err != nil {
		return nil, fmt.Errorf("can not resolve bindings: %w", err)
	}

	resourceContainer = &container{
		typ:      typ,
		name:     options.Name,
		bindings: resolvedBindings,
	}

	return resourceContainer, nil
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
	resolvedAddresses := make(map[string]containerBinding)

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

var (
	alreadyExists   = regexp.MustCompile(`API error \(409\): removal of container (\w+) is already in progress`)
	noSuchContainer = regexp.MustCompile(`No such container: (\w+)`)
)

func (r *containerRunner) Stop() error {
	for name, cb := range r.shutdownCallbacks {
		err := cb()
		if err != nil {
			r.logger.Error("shutdown callback failed for container %s: %w", name, err)
		}
	}

	for name, resource := range r.resources {
		if err := r.pool.Purge(resource); err != nil {
			if !alreadyExists.MatchString(err.Error()) && !noSuchContainer.MatchString(err.Error()) {
				return fmt.Errorf("could not stop container %s: %w", name, err)
			}

			r.logger.Debug("someone else is already stopping container %s, ignoring error %s", name, err.Error())
		}

		r.logger.Debug("stopping container %s", name)
	}

	return nil
}
