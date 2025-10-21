package env

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/ory/dockertest/v3/docker"
)

const (
	networkBridge    = "bridge"
	osLinux          = "linux"
	RunnerTypeLocal  = "local"
	RunnerTypeRemote = "remote"
)

var containerRunnerFactories = map[string]func(cfg.Config, log.Logger, *ContainerManagerSettings) (ContainerRunner, error){
	RunnerTypeLocal:  NewContainerRunnerLocal,
	RunnerTypeRemote: NewContainerRunnerRemote,
}

type ContainerRunner interface {
	RunContainer(ctx context.Context, request ContainerRequest) (*Container, error)
	Stop(ctx context.Context) error
}

type ContainerRequest struct {
	TestName             string
	ComponentType        string
	ComponentName        string
	ContainerName        string
	ContainerDescription *ComponentContainerDescription
	ExpireAfter          time.Duration
}

func (r ContainerRequest) id() string {
	return fmt.Sprintf("%s-%s", r.ComponentType, r.ComponentName)
}

type ContainerConfig struct {
	RunnerType   string
	Hostname     string
	Auth         authSettings
	Repository   string
	Tmpfs        []TmpfsSettings
	Tag          string
	Env          map[string]string
	Cmd          []string
	PortBindings PortBindings
	ExposedPorts []string
}

func (c ContainerConfig) String() string {
	return fmt.Sprintf("%s:%s", c.Repository, c.Tag)
}

type PortBindings map[string]PortBinding

type PortBinding struct {
	ContainerPort int    `json:"container_port"`
	HostPort      int    `json:"host_port"`
	Protocol      string `json:"protocol"`
}

func (b PortBinding) DockerPort() string {
	return fmt.Sprintf("%d/%s", b.ContainerPort, b.Protocol)
}

type Container struct {
	typ      string
	name     string
	bindings map[string]ContainerBinding
}

type ContainerBinding struct {
	host string
	port string
}

func (b ContainerBinding) getAddress() string {
	return fmt.Sprintf("%s:%s", b.host, b.port)
}

type authSettings struct {
	Username      string `cfg:"username"       default:""`
	Password      string `cfg:"password"       default:""`
	Email         string `cfg:"email"          default:""`
	ServerAddress string `cfg:"server_address" default:""`
}

func (a authSettings) GetAuthConfig() docker.AuthConfiguration {
	return docker.AuthConfiguration{
		Username:      a.Username,
		Password:      a.Password,
		Email:         a.Email,
		ServerAddress: a.ServerAddress,
	}
}

// IsEmpty returns true when username and password are empty as both are the minimum needed for authentication
func (a authSettings) IsEmpty() bool {
	return a.Username == "" && a.Password == ""
}
