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
	HealthCheck  func(*dockertest.Resource) error
	PrintLogs    bool
	ExpireAfter  time.Duration
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

func (d *dockerRunner) Run(name string, config containerConfig) (*dockertest.Resource, error) {
	containerName := d.getContainerName(name)

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
		return nil, fmt.Errorf("could not start %s container: %w", containerName, err)
	}

	err = resource.Expire(uint(config.ExpireAfter.Seconds()))

	if err != nil {
		return nil, fmt.Errorf("could not expire %s container: %w", containerName, err)
	}

	logger.WithFields(map[string]interface{}{
		"expire_after": config.ExpireAfter,
	}).Info("set container expiry")

	err = d.pool.Retry(func() error {
		return config.HealthCheck(resource)
	})

	if err != nil {
		return nil, fmt.Errorf("could not bring up %s container: %w", containerName, err)
	}

	d.resourcesMutex.Lock()
	d.resources = append(d.resources, resource)
	defer d.resourcesMutex.Unlock()

	if config.PrintLogs {
		d.printContainerLogs(resource)
	}

	logger.Info("container up and running")

	return resource, nil
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
		d.logger.WithFields(map[string]interface{}{
			"container": res.Container.Name,
		}).Infof("stopping container")
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
