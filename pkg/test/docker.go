package test

import (
	"fmt"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
)

type PortBinding map[string]string

type ContainerConfig struct {
	Repository   string
	Tag          string
	Env          []string
	PortBindings PortBinding
	HealthCheck  func() error
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

	resource, err := dockerPool.RunWithOptions(&dockertest.RunOptions{
		Name:         name,
		Repository:   config.Repository,
		Tag:          config.Tag,
		Env:          config.Env,
		PortBindings: bindings,
	})

	if err != nil {
		logErr(err, fmt.Sprintf("could not start %s container", name))
	}

	err = resource.Expire(60 * 60)

	if err != nil {
		logErr(err, "Could not expire resource")
	}

	err = dockerPool.Retry(config.HealthCheck)

	if err != nil {
		logErr(err, fmt.Sprintf("could not bring up %s container", name))
	}

	dockerResources = append(dockerResources, resource)
}
