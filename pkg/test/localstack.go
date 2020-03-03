package test

import (
	"errors"
	"regexp"
)

// simple logs based check for now
// it will be replaced with a proper health check http endpoint
// once it has been released in localstack
// see: https://github.com/localstack/localstack/pull/2080
func localstackHealthCheck(runner *dockerRunner, containerName string) func() error {
	return func() error {
		logs, err := runner.GetLogs(containerName)

		if err != nil {
			return err
		}

		ready, err := regexp.MatchString("Ready\\.", logs)

		if err != nil {
			return err
		}

		if !ready {
			return errors.New("localstack services not ready yet")
		}

		return nil
	}
}
