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
}

type dockerRunner struct {
	pool           *dockertest.Pool
	resources      []*dockertest.Resource
	resourcesMutex sync.Mutex
	id             string
	logger         mon.Logger
}

func newDockerRunner() *dockerRunner {
	pool, err := dockertest.NewPool("")

	if err != nil {
		log.Fatalf("could not connect to docker: %s", err)
	}

	pool.MaxWait = 2 * time.Minute

	resources := make([]*dockertest.Resource, 0)

	id := uuid.New().NewV4()

	logger := mon.NewLogger().WithChannel("docker-runner")

	return &dockerRunner{
		pool:      pool,
		resources: resources,
		id:        id,
		logger:    logger,
	}
}

func (d *dockerRunner) Run(name string, config containerConfig) {

	containerName := d.getContainerName(name)

	bindings := make(map[docker.Port][]docker.PortBinding)
	for containerPort, hostPort := range config.PortBindings {
		bindings[docker.Port(containerPort)] = []docker.PortBinding{
			{
				HostPort: hostPort,
			},
		}
	}

	d.logger.Info(fmt.Sprintf("starting container %s", containerName))
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

	err = resource.Expire(60 * 60)

	if err != nil {
		panic(fmt.Errorf("could not expire %s container: %w", containerName, err))
	}

	err = d.pool.Retry(config.HealthCheck)

	if err != nil {
		panic(fmt.Errorf("could not bring up %s container: %w", containerName, err))
	}

	d.resourcesMutex.Lock()
	d.resources = append(d.resources, resource)
	defer d.resourcesMutex.Unlock()

	if config.PrintLogs {
		d.printContainerLogs(resource)
	}

	d.logger.Info(fmt.Sprintf("container up and running %s", containerName))
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

func (d *dockerRunner) PurgeAllResources() {
	for _, res := range d.resources {
		if err := d.pool.Purge(res); err != nil {
			log.Fatalf("Could not purge resource: %s", err)
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
