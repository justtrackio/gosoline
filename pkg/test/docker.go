package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/uuid"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

type portBinding map[string]string

type containerConfig struct {
	Repository   string
	Tag          string
	Env          []string
	Cmd          []string
	PortBindings portBinding
	PortMappings map[string]*int
	HealthCheck  func() error
	PrintLogs    bool
	ExpireAfter  time.Duration
}

type dockerRunner struct {
	pool            *dockertest.Pool
	containers      []string
	containersMutex sync.Mutex
	id              string
	logger          mon.Logger
}

func newDockerRunner() *dockerRunner {
	pool, err := dockertest.NewPool("")

	if err != nil {
		log.Fatalf("could not connect to docker: %s", err)
	}

	pool.MaxWait = 2 * time.Minute

	containers := make([]string, 0)

	id := uuid.New().NewV4()

	logger := mon.NewLogger().WithChannel("docker-runner")

	return &dockerRunner{
		pool:       pool,
		id:         id,
		logger:     logger,
		containers: containers,
	}
}

func (d *dockerRunner) Run(name string, config containerConfig) error {
	containerName := d.getContainerName(name)

	d.markForCleanup(containerName)

	logger := d.logger.WithFields(map[string]interface{}{
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
	resource, err := d.pool.RunWithOptions(&dockertest.RunOptions{
		Name:         containerName,
		Repository:   config.Repository,
		Tag:          config.Tag,
		Env:          config.Env,
		Cmd:          config.Cmd,
		PortBindings: bindings,
	})

	if err != nil {
		return fmt.Errorf("could not start %s container: %w", containerName, err)
	}

	err = resource.Expire(uint(config.ExpireAfter.Seconds()))

	if err != nil {
		return fmt.Errorf("could not expire %s container: %w", containerName, err)
	}

	for containerPort, hostPort := range config.PortMappings {
		err = d.setPortMapping(resource, containerPort, hostPort)
		if err != nil {
			return err
		}
	}

	logger.WithFields(map[string]interface{}{
		"expire_after": config.ExpireAfter,
	}).Info("set container expiry")

	err = d.pool.Retry(func() error {
		return config.HealthCheck()
	})

	if err != nil {
		return fmt.Errorf("could not bring up %s container: %w", containerName, err)
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

func (d *dockerRunner) markForCleanup(containerName string) {
	d.containersMutex.Lock()
	defer d.containersMutex.Unlock()
	d.containers = append(d.containers, containerName)
}

func (d *dockerRunner) printContainerLogs(resource *dockertest.Resource) error {
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

func (d *dockerRunner) RemoveAllContainers() {
	for _, containerName := range d.containers {
		d.logger.WithFields(map[string]interface{}{
			"container": containerName,
		}).Infof("stopping container")
		if err := d.pool.RemoveContainerByName(containerName); err != nil {
			d.logger.Warn("could not remove container %s: %w", containerName, err)
		}
	}
}

func (d *dockerRunner) getContainerName(name string) string {
	return fmt.Sprintf("%s_%s", name, d.id)
}

func (d *dockerRunner) setPortMapping(resource *dockertest.Resource, containerPort string, hostPort *int) error {
	dockerPort := docker.Port(containerPort)

	port, err := strconv.Atoi(resource.Container.NetworkSettings.Ports[dockerPort][0].HostPort)
	if err != nil {
		return err
	}

	d.logger.WithFields(map[string]interface{}{
		"container": resource.Container.Name[1:],
	}).Infof("set port mapping %s:%d", containerPort, port)

	*hostPort = port

	return nil
}
