package env

import (
	"context"
	"strconv"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

const RunnerTypeExternal = "external"

func init() {
	containerRunnerFactories[RunnerTypeExternal] = NewContainerRunnerExternal
}

var _ ContainerRunner = (*containerRunnerExternal)(nil)

type containerRunnerExternal struct {
	logger log.Logger
}

// NewContainerRunnerExternal creates a runner that connects to external (pre-existing) containers
// rather than starting new ones. This is useful for CI environments where containers are
// provided as external services.
func NewContainerRunnerExternal(_ cfg.Config, logger log.Logger, _ *ContainerManagerSettings) (ContainerRunner, error) {
	return &containerRunnerExternal{
		logger: logger.WithChannel("container-runner-external"),
	}, nil
}

func (r *containerRunnerExternal) RunContainer(ctx context.Context, request ContainerRequest) (*Container, error) {
	config := request.ContainerDescription.ContainerConfig
	r.logger.Debug(ctx, "using external container %s %s", request.ComponentType, request.ComponentName)

	container := &Container{
		typ:      request.ComponentType,
		name:     request.ComponentName,
		bindings: make(map[string]ContainerBinding),
	}

	// Use the external host from the container config (set by the component factory)
	host := config.ExternalHost
	if host == "" {
		host = "127.0.0.1"
	}

	// Map the configured port bindings to the external host/port
	for name, binding := range config.PortBindings {
		// Use the configured port (ContainerPort contains the external port for external containers)
		container.bindings[name] = ContainerBinding{
			host: host,
			port: strconv.Itoa(binding.ContainerPort),
		}
	}

	return container, nil
}

func (r *containerRunnerExternal) Stop(_ context.Context) error {
	// External containers are not managed by us, so nothing to stop
	return nil
}
