package test

import (
	"fmt"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"log"
	"os"
)

type PortBinding map[string]string

type ContainerConfig struct {
	Repository   string
	Tag          string
	Env          []string
	Cmd          []string
	PortBindings PortBinding
	HealthCheck  func() error
	OnDestroy    func()
	PrintLogs    bool
}

func runContainer(name string, config ContainerConfig) {
	err := dockerPool.RemoveContainerByName(name)

	if err != nil {
		logErr(err, fmt.Sprintf("could not remove existing %s container", name))
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
	resource, err := dockerPool.RunWithOptions(&dockertest.RunOptions{
		Name:         name,
		Repository:   config.Repository,
		Tag:          config.Tag,
		Env:          config.Env,
		Cmd:          config.Cmd,
		PortBindings: bindings,
	})

	if err != nil {
		logErr(err, fmt.Sprintf("could not start %s container", name))
	}

	err = resource.Expire(60 * 60)

	if err != nil {
		logErr(err, fmt.Sprintf("could not expire container %s", name))
	}

	err = dockerPool.Retry(config.HealthCheck)

	if err != nil {
		logErr(err, fmt.Sprintf("could not bring up %s container", name))
	}

	dockerResources = append(dockerResources, resource)
	configs = append(configs, &config)

	if config.PrintLogs {
		printContainerLogs(resource)
	}

}

func printContainerLogs(resource *dockertest.Resource) {
	err := dockerPool.Client.Logs(docker.LogsOptions{
		Container:    resource.Container.ID,
		OutputStream: os.Stdout,
		ErrorStream:  os.Stderr,
		Stdout:       true,
		Stderr:       true,
	})

	if err != nil {
		logErr(err, fmt.Sprintf("could not print docker logs for container: %s", resource.Container.Name))
	}
}
