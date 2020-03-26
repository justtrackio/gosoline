package test

import (
	"bytes"
	"fmt"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/uuid"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"log"
	"os"
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

	id := uuid.New().NewV4()

	logger := mon.NewLogger().WithChannel("docker-runner")

	containers := make([]string, 0)

	return &dockerRunner{
		pool:       pool,
		id:         id,
		logger:     logger,
		containers: containers,
	}
}

func (d *dockerRunner) Run(name string, config containerConfig) {

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
		panic(fmt.Errorf("could not start %s container: %w", containerName, err))
	}

	err = resource.Expire(uint(config.ExpireAfter.Seconds()))

	if err != nil {
		panic(fmt.Errorf("could not expire %s container: %w", containerName, err))
	}

	logger.WithFields(map[string]interface{}{
		"expire_after": config.ExpireAfter,
	}).Info("set container expiry")

	err = d.pool.Retry(config.HealthCheck)

	if err != nil {
		panic(fmt.Errorf("could not bring up %s container: %w", containerName, err))
	}

	if config.PrintLogs {
		d.printContainerLogs(resource)
	}

	logger.Info("container up and running")
}

func (d *dockerRunner) markForCleanup(containerName string) {
	d.containersMutex.Lock()
	defer d.containersMutex.Unlock()
	d.containers = append(d.containers, containerName)
}

func (d *dockerRunner) printContainerLogs(resource *dockertest.Resource) {
	err := d.pool.Client.Logs(docker.LogsOptions{
		Container:    resource.Container.ID,
		OutputStream: os.Stdout,
		ErrorStream:  os.Stderr,
		Stdout:       true,
		Stderr:       true,
	})

	if err != nil {
		panic(fmt.Errorf("could not print docker logs for container %s: %w", resource.Container.Name, err))
	}
}

func (d *dockerRunner) RemoveAllContainers() {
	for _, container := range d.containers {
		d.logger.WithFields(map[string]interface{}{
			"container": container,
		}).Infof("stopping container")
		if err := d.pool.RemoveContainerByName(container); err != nil {
			d.logger.Warn("could not remove container %s: %w", container, err)
		}
	}
}

func (d *dockerRunner) GetLogs(name string) (string, error) {
	logs := bytes.NewBufferString("")

	containerName := d.getContainerName(name)

	err := d.pool.Client.Logs(docker.LogsOptions{
		Container:    containerName,
		OutputStream: logs,
		Stdout:       true,
		Stderr:       true,
	})

	if err != nil {
		return logs.String(), err
	}

	return logs.String(), nil
}

func (d *dockerRunner) getContainerName(name string) string {
	return fmt.Sprintf("%s_%s", name, d.id)
}
