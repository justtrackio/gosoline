package test

import (
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
		logErr(err, "could not remove existing dynamoDb container")
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
		logErr(err, "could not start dynamoDb container")
	}

	err = resource.Expire(60 * 60)

	if err != nil {
		logErr(err, "Could not expire resource")
	}

	err = dockerPool.Retry(config.HealthCheck)

	if err != nil {
		logErr(err, "could not bring up dynamoDb container")
	}

	dockerResources = append(dockerResources, resource)
}
