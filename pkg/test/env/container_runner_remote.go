package env

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/uuid"
)

var _ ContainerRunner = (*containerRunnerRemote)(nil)

type ContainerRunnerRemoteSettings struct {
	Endpoint   string                                  `cfg:"endpoint"    default:"http://localhost:8890"`
	PoolId     string                                  `cfg:"pool_id"`
	HttpClient ContainerRunnerRemoteHttpClientSettings `cfg:"http_client"`
}

type ContainerRunnerRemoteHttpClientSettings struct {
	Timeout       time.Duration `cfg:"timeout"     default:"30s"`
	RetryCount    int           `cfg:"retry_count" default:"32"`
	RetryWaitTime time.Duration `cfg:"retry_wait"  default:"3s"`
}

type ContainerStartInput struct {
	PoolId        string        `json:"pool_id"`
	TestId        string        `json:"test_id"`
	TestName      string        `json:"test_name"`
	ComponentType string        `json:"component_type"`
	ComponentName string        `json:"component_name"`
	ContainerName string        `json:"container_name"`
	Spec          ContainerSpec `json:"spec"`
	ExpireAfter   time.Duration `json:"expire_after"`
}

type ContainerSpec struct {
	Repository   string            `json:"repository"`
	Tag          string            `json:"tag"`
	Env          map[string]string `json:"env"`
	Cmd          []string          `json:"cmd"`
	PortBindings PortBindings      `json:"port_bindings"`
}

type ContainerRefreshInput struct {
	PoolId   string        `json:"pool_id"`
	TestId   string        `json:"test_id"`
	Duration time.Duration `json:"duration"`
}

type ContainerStopInput struct {
	PoolId string `json:"pool_id"`
	TestId string `json:"test_id"`
}

type containerRunnerRemote struct {
	logger          log.Logger
	client          *resty.Client
	managerSettings *ContainerManagerSettings
	runnerSettings  *ContainerRunnerRemoteSettings
	testId          string
}

func NewContainerRunnerRemote(config cfg.Config, logger log.Logger, managerSettings *ContainerManagerSettings) (ContainerRunner, error) {
	runnerSettings := &ContainerRunnerRemoteSettings{}
	if err := config.UnmarshalKey("test.container_manager.runner.remote", runnerSettings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal container runner settings: %w", err)
	}

	if runnerSettings.PoolId == "" {
		runnerSettings.PoolId = uuid.New().NewV4()[:8]
	}

	logger = logger.WithChannel("container-runner-remote")

	client := resty.New().SetBaseURL(runnerSettings.Endpoint)
	client.SetTimeout(runnerSettings.HttpClient.Timeout)
	client.SetRetryCount(runnerSettings.HttpClient.RetryCount)
	client.SetRetryWaitTime(runnerSettings.HttpClient.RetryWaitTime)
	client.AddRetryCondition(func(r *resty.Response, err error) bool {
		if exec.IsConnectionError(err) {
			return true
		}

		if errors.Is(err, context.DeadlineExceeded) {
			return true
		}

		return false
	})

	return &containerRunnerRemote{
		logger:          logger,
		client:          client,
		managerSettings: managerSettings,
		runnerSettings:  runnerSettings,
		testId:          uuid.New().NewV4(),
	}, nil
}

func (r *containerRunnerRemote) RunContainer(ctx context.Context, request ContainerRequest) (*Container, error) {
	config := request.ContainerDescription.ContainerConfig
	r.logger.Debug(ctx, "run container %s %s:%s", request.ComponentType, config.Repository, config.Tag)

	input := &ContainerStartInput{
		PoolId:        r.runnerSettings.PoolId,
		TestId:        r.testId,
		TestName:      request.TestName,
		ComponentType: request.ComponentType,
		ComponentName: request.ComponentName,
		ContainerName: request.ContainerName,
		Spec: ContainerSpec{
			Repository:   config.Repository,
			Tag:          config.Tag,
			Env:          config.Env,
			Cmd:          config.Cmd,
			PortBindings: config.PortBindings,
		},
		ExpireAfter: request.ExpireAfter,
	}

	var err error
	var resp *resty.Response
	bindings := make(map[string]string)

	req := r.client.R().SetBody(input).SetResult(&bindings)
	if resp, err = req.Post("/run"); err != nil {
		return nil, fmt.Errorf("could not start container: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("got response code %d, could not start container: %s", resp.StatusCode(), resp.String())
	}

	container := &Container{
		typ:      request.ComponentType,
		name:     request.ComponentName,
		bindings: make(map[string]ContainerBinding),
	}

	for name, hostPort := range bindings {
		host, port, err := net.SplitHostPort(hostPort)
		if err != nil {
			return nil, fmt.Errorf("could not split host and port: %w", err)
		}

		container.bindings[name] = ContainerBinding{
			host: host,
			port: port,
		}
	}

	return container, nil
}

func (r *containerRunnerRemote) ExtendLifetime(ctx context.Context, duration time.Duration) error {
	var err error
	var resp *resty.Response

	input := &ContainerRefreshInput{
		PoolId:   r.runnerSettings.PoolId,
		TestId:   r.testId,
		Duration: duration,
	}

	req := r.client.R().SetBody(input)
	if resp, err = req.Post("/extend"); err != nil {
		return fmt.Errorf("could not extend container: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("got response code %d, could not extend containers: %s", resp.StatusCode(), resp.String())
	}

	return nil
}

func (r *containerRunnerRemote) Stop(ctx context.Context) error {
	var err error
	var resp *resty.Response

	input := &ContainerStopInput{
		PoolId: r.runnerSettings.PoolId,
		TestId: r.testId,
	}

	req := r.client.R().SetBody(input)
	if resp, err = req.Post("/stop"); err != nil {
		return fmt.Errorf("could not stop container: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("got response code %d, could not stop container: %s", resp.StatusCode(), resp.String())
	}

	return nil
}
