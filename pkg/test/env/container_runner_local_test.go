package env

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainerRunnerLocalCopyFilesWithoutFiles(t *testing.T) {
	require.NoError(t, new(containerRunnerLocal).copyFiles(t.Context(), nil, nil))
}

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

func TestArchiveContainerFiles(t *testing.T) {
	data := []byte{0, 0xff, 0x80, '\n'}
	filename := filepath.Join(t.TempDir(), "binary.gz")
	require.NoError(t, os.WriteFile(filename, data, 0o600))
	archive, err := archiveContainerFiles([]ContainerFile{{Source: filename, Target: "/tmp/fixtures/binary.gz"}})
	require.NoError(t, err)
	reader := tar.NewReader(bytes.NewReader(archive))
	header, err := reader.Next()
	require.NoError(t, err)
	require.Equal(t, "tmp/fixtures/binary.gz", header.Name)
	require.Equal(t, int64(0o644), header.Mode)
	actual, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, data, actual)
	_, err = reader.Next()
	require.ErrorIs(t, err, io.EOF)
}

func TestArchiveContainerFilesRejectsInvalidPaths(t *testing.T) {
	for _, target := range []string{"relative.gz", "/", "/tmp/../response.gz"} {
		t.Run(target, func(t *testing.T) {
			_, err := archiveContainerFiles([]ContainerFile{{Source: "unused", Target: target}})
			require.ErrorContains(t, err, "must be a clean absolute file path")
		})
	}
}

func TestArchiveContainerFilesRejectsMissingSource(t *testing.T) {
	_, err := archiveContainerFiles([]ContainerFile{{Source: filepath.Join(t.TempDir(), "missing.gz"), Target: "/tmp/response.gz"}})
	require.ErrorIs(t, err, os.ErrNotExist)
}
