package env

import (
	"testing"

	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/assert"
)

func TestContainerRunnerLocalGetAuthConfig(t *testing.T) {
	ecr := authSettings{Username: "AWS", Password: "test-token", ServerAddress: "registry.example.com"}
	hub := authSettings{Username: "hub-user", Password: "hub-token", ServerAddress: "https://index.docker.io/v1/"}
	explicit := authSettings{Username: "component-user", Password: "component-token"}

	for _, test := range []struct {
		name       string
		repository string
		runner     authSettings
		component  authSettings
		want       docker.AuthConfiguration
	}{
		{name: "matching private registry", repository: "registry.example.com/mmock", runner: ecr, want: ecr.GetAuthConfig()},
		{name: "public image ignores ECR credentials", repository: "jordimartin/mmock", runner: ecr},
		{name: "other private registry ignores ECR credentials", repository: "other.example.com/mmock", runner: ecr},
		{name: "Docker Hub URL matches shorthand", repository: "jordimartin/mmock", runner: hub, want: hub.GetAuthConfig()},
		{name: "Docker Hub URL matches explicit registry", repository: "docker.io/jordimartin/mmock", runner: hub, want: hub.GetAuthConfig()},
		{name: "Docker Hub URL matches registry alias", repository: "registry-1.docker.io/jordimartin/mmock", runner: hub, want: hub.GetAuthConfig()},
		{name: "registry port must match", repository: "registry.example.com:5000/mmock", runner: ecr},
		{name: "component credentials override runner", repository: "jordimartin/mmock", runner: ecr, component: explicit, want: explicit.GetAuthConfig()},
		{name: "unscoped runner credentials remain supported", repository: "jordimartin/mmock", runner: explicit, want: explicit.GetAuthConfig()},
		{name: "anonymous", repository: "jordimartin/mmock"},
	} {
		t.Run(test.name, func(t *testing.T) {
			runner := &containerRunnerLocal{runnerSettings: &ContainerRunnerLocalSettings{Auth: test.runner}}
			assert.Equal(t, test.want, runner.getAuthConfig(test.component, test.repository))
		})
	}
}
