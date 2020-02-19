package test

import (
	"bytes"
	"errors"
	"github.com/ory/dockertest/docker"
	"regexp"
)

// simple logs based check for now
// it will be replaced with a proper health check http endpoint
// once it has been released in localstack
// see: https://github.com/localstack/localstack/pull/2080
func localstackHealthCheck(containerName string) func() error {
	return func() error {
		logs := bytes.NewBufferString("")

		err := dockerPool.Client.Logs(docker.LogsOptions{
			Container:    containerName,
			OutputStream: logs,
			Stdout:       true,
			Stderr:       true,
		})

		if err != nil {
			return err
		}

		ready, err := regexp.MatchString("Ready\\.", logs.String())

		if err != nil {
			return err
		}

		if !ready {
			return errors.New("localstack services not ready yet")
		}

		return nil
	}
}
