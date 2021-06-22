package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/test/env"
	"github.com/applike/gosoline/pkg/uuid"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"github.com/spf13/cast"
	goLog "log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

type portBindingLegacy map[string]string

type portMappingLegacy map[string]*int

type hostMappingLegacy struct {
	dialPort *int
	setHost  *string
}

type containerConfigLegacy struct {
	Repository   string
	Tmpfs        map[string]interface{}
	Tag          string
	Env          []string
	Cmd          []string
	PortBindings portBindingLegacy
	PortMappings portMappingLegacy
	HostMapping  hostMappingLegacy
	HealthCheck  func() error
	PrintLogs    bool
	ExpireAfter  time.Duration
}

type authSettingsLegacy struct {
	Username      string `cfg:"username" default:""`
	Password      string `cfg:"password" default:""`
	Email         string `cfg:"email" default:""`
	ServerAddress string `cfg:"server_address" default:""`
}

func (a authSettingsLegacy) GetAuthConfig() docker.AuthConfiguration {
	return docker.AuthConfiguration{
		Username:      a.Username,
		Password:      a.Password,
		Email:         a.Email,
		ServerAddress: a.ServerAddress,
	}
}

type dockerRunnerLegacy struct {
	pool            *dockertest.Pool
	containers      []string
	containersMutex sync.Mutex
	id              string
	logger          log.Logger
	auth            *authSettingsLegacy
}

func NewDockerRunnerLegacy(config cfg.Config) *dockerRunnerLegacy {
	pool, err := dockertest.NewPool("")

	if err != nil {
		goLog.Fatalf("could not connect to docker: %s", err)
	}

	pool.MaxWait = 2 * time.Minute
	containers := make([]string, 0)
	id := uuid.New().NewV4()

	gosoLogger, err := env.NewConsoleLogger()
	if err != nil {
		goLog.Fatalf("could not create logger: %s", err)
	}

	logger := gosoLogger.WithChannel("docker-runner")

	auth := &authSettingsLegacy{}
	// take the auth config from the new container runner so we don't have to specify it twice
	config.UnmarshalKey("test.container_runner.auth", auth)

	return &dockerRunnerLegacy{
		pool:       pool,
		id:         id,
		logger:     logger,
		containers: containers,
		auth:       auth,
	}
}

func (d *dockerRunnerLegacy) Run(name string, config *containerConfigLegacy) error {
	containerName := d.getContainerName(name)

	d.markForCleanup(containerName)

	logger := d.logger.WithFields(log.Fields{
		"container": containerName,
	})

	bindings := make(map[docker.Port][]docker.PortBinding)
	for containerPort, hostPort := range config.PortBindings {
		bindings[docker.Port(containerPort)] = []docker.PortBinding{
			{
				HostPort: hostPort,
			},
		}
	}

	logger.Info("starting container")

	runOptions := &dockertest.RunOptions{
		Name:         containerName,
		Repository:   config.Repository,
		Tag:          config.Tag,
		Env:          config.Env,
		Cmd:          config.Cmd,
		PortBindings: bindings,
		Auth:         d.auth.GetAuthConfig(),
	}
	hostConfigs := make([]func(hc *docker.HostConfig), 0)

	if len(config.Tmpfs) > 0 {
		tmpfs, err := cast.ToStringMapStringE(config.Tmpfs)

		if err != nil {
			return fmt.Errorf("can not cast tmpfs config to map[string]string: %w", err)
		}

		hostConfigs = append(hostConfigs, func(hc *docker.HostConfig) {
			hc.Tmpfs = tmpfs
		})
	}

	resource, err := d.pool.RunWithOptions(runOptions, hostConfigs...)

	if err != nil {
		return fmt.Errorf("could not start container %s: %w", containerName, err)
	}

	err = resource.Expire(uint(config.ExpireAfter.Seconds()))

	if err != nil {
		return fmt.Errorf("could not expire container %s: %w", containerName, err)
	}

	logger.WithFields(log.Fields{
		"expire_after": config.ExpireAfter,
	}).Info("set container expiry")

	for containerPort, hostPort := range config.PortMappings {
		err = d.setPortMapping(resource, containerPort, hostPort)
		if err != nil {
			return err
		}
	}

	err = d.pool.Retry(func() error {
		return d.setHostMapping(resource, config.HostMapping)
	})

	if err != nil {
		return fmt.Errorf("could not set host mapping for container %s: %w", containerName, err)
	}

	err = d.pool.Retry(func() error {
		return config.HealthCheck()
	})

	if err != nil {
		return fmt.Errorf("could not check health of container %s: %w", containerName, err)
	}

	if config.PrintLogs {
		err := d.printContainerLogs(resource)
		if err != nil {
			return err
		}
	}

	logger.Info("container up and running")

	return nil
}

func (d *dockerRunnerLegacy) getContainerName(name string) string {
	return fmt.Sprintf("%s_%s", name, d.id)
}

func (d *dockerRunnerLegacy) markForCleanup(containerName string) {
	d.containersMutex.Lock()
	defer d.containersMutex.Unlock()
	d.containers = append(d.containers, containerName)
}

func (d *dockerRunnerLegacy) setPortMapping(resource *dockertest.Resource, containerPort string, hostPort *int) error {
	port, err := strconv.Atoi(resource.GetPort(containerPort))
	if err != nil {
		return err
	}

	d.logger.WithFields(log.Fields{
		"container":    resource.Container.Name[1:],
		"port_mapping": fmt.Sprintf("%s:%d", containerPort, port),
	}).Info("set port mapping")

	*hostPort = port

	return nil
}

func (d *dockerRunnerLegacy) printContainerLogs(resource *dockertest.Resource) error {
	err := d.pool.Client.Logs(docker.LogsOptions{
		Container:    resource.Container.ID,
		OutputStream: os.Stdout,
		ErrorStream:  os.Stderr,
		Stdout:       true,
		Stderr:       true,
	})

	if err != nil {
		return fmt.Errorf("could not print docker logs for container %s: %w", resource.Container.Name, err)
	}

	return nil
}

func (d *dockerRunnerLegacy) RemoveAllContainers() {
	for _, containerName := range d.containers {
		d.logger.WithFields(log.Fields{
			"container": containerName,
		}).Info("stopping container")
		if err := d.pool.RemoveContainerByName(containerName); err != nil {
			d.logger.Warn("could not remove container %s: %w", containerName, err)
		}
	}
}

func (d *dockerRunnerLegacy) setHostMapping(resource *dockertest.Resource, mapping hostMappingLegacy) error {
	timeout := time.Duration(100) * time.Millisecond
	gatewayIp := resource.Container.NetworkSettings.Networks["bridge"].Gateway

	addresses := []string{gatewayIp, "127.0.0.1"}
	for _, ip := range addresses {
		address := fmt.Sprintf("%s:%d", ip, *mapping.dialPort)
		if d.isReachable(address, timeout) {
			d.logger.WithFields(log.Fields{
				"container": resource.Container.Name[1:],
				"ip":        ip,
			}).Info("set host mapping")

			*mapping.setHost = ip

			return nil
		}
	}

	return fmt.Errorf("could not establish a connection with the container")
}

func (d *dockerRunnerLegacy) isReachable(address string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", address, timeout)
	defer func() {
		if conn == nil {
			return
		}

		err := conn.Close()

		if err != nil {
			d.logger.Error("failed to close connection: %w", err)
		}
	}()

	return err == nil
}
