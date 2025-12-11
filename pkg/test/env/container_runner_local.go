package env

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/uuid"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

var _ ContainerRunner = &containerRunnerLocal{}

var (
	alreadyExists   = regexp.MustCompile(`API error \(409\): removal of container (\w+) is already in progress`)
	noSuchContainer = regexp.MustCompile(`No such container: (\w+)`)
)

type ContainerRunnerLocalSettings struct {
	Endpoint string       `cfg:"endpoint"`
	Auth     authSettings `cfg:"auth"`
}

type containerRunnerLocal struct {
	logger          log.Logger
	pool            *dockertest.Pool
	id              string
	resources       map[string]*dockertest.Resource
	resourcesLck    sync.Mutex
	managerSettings *ContainerManagerSettings
	runnerSettings  *ContainerRunnerLocalSettings
}

func NewContainerRunnerLocal(config cfg.Config, logger log.Logger, managerSettings *ContainerManagerSettings) (ContainerRunner, error) {
	id := uuid.New().NewV4()
	logger = logger.WithChannel("container-runner-local")

	runnerSettings := &ContainerRunnerLocalSettings{}
	if err := config.UnmarshalKey("test.container_manager.runner.local", runnerSettings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal container runner settings: %w", err)
	}

	pool, err := dockertest.NewPool(runnerSettings.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("can not create docker pool: %w", err)
	}

	// do this here, if we let the call to `pool.Retry` do this, we trigger a data race (not bad in this case, but annoying
	// for the race detector)
	if pool.MaxWait == 0 {
		pool.MaxWait = time.Minute
	}

	return &containerRunnerLocal{
		logger:          logger,
		pool:            pool,
		id:              id,
		resources:       make(map[string]*dockertest.Resource),
		resourcesLck:    sync.Mutex{},
		managerSettings: managerSettings,
		runnerSettings:  runnerSettings,
	}, nil
}

func (r *containerRunnerLocal) RunContainer(ctx context.Context, request ContainerRequest) (*Container, error) {
	var container *Container
	var err error

	if err = r.pullContainerImage(request.ContainerDescription); err != nil {
		return nil, fmt.Errorf("could not pull container image %q: %w", request.ContainerDescription.ContainerConfig, err)
	}

	config := request.ContainerDescription.ContainerConfig
	containerFqn := fmt.Sprintf("%s-%s-%s-%s", r.managerSettings.NamePrefix, r.id, request.id(), request.ContainerName)

	r.logger.Debug(ctx, "run container %s %s:%s %s", request.ComponentType, config.Repository, config.Tag, containerFqn)

	bindings := make(map[docker.Port][]docker.PortBinding)

	for _, binding := range config.PortBindings {
		bindings[docker.Port(binding.DockerPort())] = []docker.PortBinding{
			{
				HostPort: fmt.Sprint(binding.HostPort),
			},
		}
	}

	containerAuth := config.Auth
	authConfig := r.getAuthConfig(containerAuth)

	envs := make([]string, 0, len(config.Env))
	for k, v := range config.Env {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}

	runOptions := &dockertest.RunOptions{
		Hostname:     config.Hostname,
		Name:         containerFqn,
		Repository:   config.Repository,
		Tag:          config.Tag,
		Env:          envs,
		Cmd:          config.Cmd,
		PortBindings: bindings,
		ExposedPorts: config.ExposedPorts,
		Auth:         authConfig,
	}

	resourceId := fmt.Sprintf("%s-%s", request.id(), request.ContainerName)
	if container, err = r.runContainer(request, runOptions, resourceId); err != nil {
		return nil, err
	}

	return container, nil
}

func (r *containerRunnerLocal) runContainer(request ContainerRequest, options *dockertest.RunOptions, resourceId string) (*Container, error) {
	var resourceContainer *Container

	config := request.ContainerDescription.ContainerConfig
	tmpfsConfig := r.getTmpfsConfig(config.Tmpfs)

	resource, err := r.pool.RunWithOptions(options, func(hc *docker.HostConfig) {
		hc.AutoRemove = true
		hc.Tmpfs = tmpfsConfig
	})
	if err != nil {
		return nil, fmt.Errorf("can not run container %s: %w", resourceId, err)
	}

	r.resourcesLck.Lock()
	r.resources[resourceId] = resource
	r.resourcesLck.Unlock()

	if err = r.expireAfter(resource, request.ExpireAfter); err != nil {
		return nil, fmt.Errorf("could not set expiry on container %s: %w", options.Name, err)
	}

	resolvedBindings, err := r.resolveBindings(resource, config.PortBindings)
	if err != nil {
		return nil, fmt.Errorf("can not resolve bindings: %w", err)
	}

	// Resolve internal bindings for container-to-container communication
	internalBindings := r.resolveInternalBindings(resource, config.PortBindings)

	resourceContainer = &Container{
		typ:              request.ComponentType,
		name:             options.Name,
		bindings:         resolvedBindings,
		internalBindings: internalBindings,
	}

	return resourceContainer, nil
}

func (r *containerRunnerLocal) pullContainerImage(description *ComponentContainerDescription) error {
	config := description.ContainerConfig
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

	containerAuth := description.ContainerConfig.Auth
	authConfig := r.getAuthConfig(containerAuth)

	err = r.pool.Client.PullImage(pullImageOptions, authConfig)
	if err != nil {
		return fmt.Errorf("could not pull image %q: %w", imageName, err)
	}

	return nil
}

func (r *containerRunnerLocal) getAuthConfig(containerAuth authSettings) docker.AuthConfiguration {
	if !containerAuth.IsEmpty() {
		return containerAuth.GetAuthConfig()
	}

	return r.runnerSettings.Auth.GetAuthConfig()
}

func (r *containerRunnerLocal) getTmpfsConfig(settings []TmpfsSettings) map[string]string {
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

func (r *containerRunnerLocal) expireAfter(resource *dockertest.Resource, expireAfter time.Duration) error {
	if err := resource.Expire(uint(expireAfter.Seconds())); err != nil {
		return err
	}

	return nil
}

func (r *containerRunnerLocal) resolveBindings(resource *dockertest.Resource, bindings PortBindings) (map[string]ContainerBinding, error) {
	var err error
	resolvedAddresses := make(map[string]ContainerBinding)

	for name, binding := range bindings {
		if resolvedAddresses[name], err = r.resolveBinding(resource, binding); err != nil {
			return nil, fmt.Errorf("failed to resolve binding: %w", err)
		}
	}

	return resolvedAddresses, nil
}

func (r *containerRunnerLocal) resolveBinding(resource *dockertest.Resource, binding PortBinding) (ContainerBinding, error) {
	var err error
	var hostPort string
	var address ContainerBinding

	err = r.pool.Retry(func() error {
		if hostPort = resource.GetHostPort(binding.DockerPort()); hostPort == "" {
			return fmt.Errorf("port is not ready yet")
		}

		return nil
	})
	if err != nil {
		return address, fmt.Errorf("can not resolve binding for port %s: %w", binding.DockerPort(), err)
	}

	if address.host, address.port, err = net.SplitHostPort(hostPort); err != nil {
		return address, fmt.Errorf("could not split hostPort into host and port: %w", err)
	}

	// On non-linux environments, the docker containers cannot be reached via the bridge network from outside the network.
	if runtime.GOOS == osLinux {
		if network, ok := resource.Container.NetworkSettings.Networks[networkBridge]; ok {
			address.host = network.Gateway
		}
	}

	return address, nil
}

// resolveInternalBindings returns bindings using the container's internal Docker IP address.
// These should be used for container-to-container communication (e.g., toxiproxy → mysql).
func (r *containerRunnerLocal) resolveInternalBindings(resource *dockertest.Resource, bindings PortBindings) map[string]ContainerBinding {
	internalBindings := make(map[string]ContainerBinding)

	network, ok := resource.Container.NetworkSettings.Networks[networkBridge]
	if !ok || network.IPAddress == "" {
		// No internal network available, return empty map
		return internalBindings
	}

	for name, binding := range bindings {
		internalBindings[name] = ContainerBinding{
			host: network.IPAddress,
			port: fmt.Sprintf("%d", binding.ContainerPort),
		}
	}

	return internalBindings
}

func (r *containerRunnerLocal) Stop(ctx context.Context) error {
	r.resourcesLck.Lock()
	defer r.resourcesLck.Unlock()

	for name, resource := range r.resources {
		if err := r.pool.Purge(resource); err != nil {
			if !alreadyExists.MatchString(err.Error()) && !noSuchContainer.MatchString(err.Error()) {
				return fmt.Errorf("could not stop container %s: %w", name, err)
			}

			r.logger.Debug(ctx, "someone else is already stopping container %s, ignoring error %s", name, err.Error())
		}

		r.logger.Debug(ctx, "stopping container %s", name)
	}

	return nil
}
