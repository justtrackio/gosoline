package test

import (
	"bytes"
	"fmt"
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
	OnDestroy    func()
	PrintLogs    bool
}

type dockerRunner struct {
	pool           *dockertest.Pool
	resources      []*dockertest.Resource
	resourcesMutex sync.Mutex
}

func newDockerRunner() *dockerRunner {
	pool, err := dockertest.NewPool("")

	if err != nil {
		log.Fatalf("could not connect to docker: %s", err)
	}

	pool.MaxWait = 2 * time.Minute

	resources := make([]*dockertest.Resource, 0)

	return &dockerRunner{
		pool:      pool,
		resources: resources,
	}
}

func (d *dockerRunner) Run(name string, config containerConfig) {

	err := d.pool.RemoveContainerByName(name)

	if err != nil {
		panic(fmt.Errorf("could not remove existing %s container : %w", name, err))
	}

	bindings := make(map[docker.Port][]docker.PortBinding)
	for containerPort, hostPort := range config.PortBindings {
		bindings[docker.Port(containerPort)] = []docker.PortBinding{
			{
				HostPort: hostPort,
			},
		}
	}

	log.Println(fmt.Sprintf("starting container %s", name))
	resource, err := d.pool.RunWithOptions(&dockertest.RunOptions{
		Name:         name,
		Repository:   config.Repository,
		Tag:          config.Tag,
		Env:          config.Env,
		Cmd:          config.Cmd,
		PortBindings: bindings,
	})

	if err != nil {
		panic(fmt.Errorf("could not start %s container : %w", name, err))
	}

	err = resource.Expire(60 * 60)

	if err != nil {
		panic(fmt.Errorf("could not expire %s container : %w", name, err))
	}

	err = d.pool.Retry(config.HealthCheck)

	if err != nil {
		panic(fmt.Errorf("could not bring up %s container : %w", name, err))
	}

	d.resourcesMutex.Lock()
	d.resources = append(d.resources, resource)
	defer d.resourcesMutex.Unlock()

	if config.PrintLogs {
		d.printContainerLogs(resource)
	}
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
		panic(fmt.Errorf("could not print docker logs for container %s : %w", resource.Container.Name, err))
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

	err := d.pool.Client.Logs(docker.LogsOptions{
		Container:    name,
		OutputStream: logs,
		Stdout:       true,
		Stderr:       true,
	})

	if err != nil {
		return logs.String(), err
	}

	return logs.String(), nil
}

func (d *dockerRunner) GetIpAddress(name string) string {
	container, err := d.pool.Client.InspectContainer(name)

	if err != nil {
		panic(fmt.Errorf("could not inspect container: %w", err))
	}

	return container.NetworkSettings.Networks["bridge"].IPAddress
}
