package env

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPullContainerImageDigest(t *testing.T) {
	const digest = "sha256:e0b292df4dc9559b7b1bf4f51bdda4bb8e269276236438cd03cae661260ea8a9"
	ecr := authSettings{Username: "AWS", Password: "ecr-token", ServerAddress: "registry.example.com"}
	hub := authSettings{Username: "hub-user", Password: "hub-token"}
	for _, tc := range []struct {
		name       string
		repository string
		tag        string
		runner     authSettings
		image      authSettings
		want       docker.AuthConfiguration
	}{
		{name: "version", repository: "floci/floci", tag: "2.3.0"},
		{name: "digest", repository: "floci/floci", tag: "2.3.0@" + digest},
		{name: "public image with private fallback", repository: "floci/floci", tag: "2.3.0@" + digest, runner: ecr},
		{name: "private image with matching fallback", repository: "registry.example.com/mysql", tag: "2.3.0@" + digest, runner: ecr, want: ecr.GetAuthConfig()},
		{name: "explicit image auth", repository: "floci/floci", tag: "2.3.0@" + digest, runner: ecr, image: hub, want: hub.GetAuthConfig()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var pulled atomic.Bool
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					http.Error(w, "image not found", http.StatusNotFound)

					return
				}
				pulled.Store(true)
				assert.Equal(t, "/images/create", r.URL.Path)
				assert.Equal(t, tc.repository, r.URL.Query().Get("fromImage"))
				expectedTag := tc.tag
				if tc.tag != "2.3.0" {
					expectedTag = digest
				}
				assert.Equal(t, expectedTag, r.URL.Query().Get("tag"))
				var auth docker.AuthConfiguration
				encoded, err := base64.URLEncoding.DecodeString(r.Header.Get("X-Registry-Auth"))
				if assert.NoError(t, err) {
					assert.NoError(t, json.Unmarshal(encoded, &auth))
				}
				assert.Equal(t, tc.want, auth)
				_, err = fmt.Fprintln(w, `{"status":"pulled"}`)
				assert.NoError(t, err)
			}))
			t.Cleanup(server.Close)
			client, err := docker.NewClient(server.URL)
			require.NoError(t, err)
			runner := &containerRunnerLocal{pool: &dockertest.Pool{Client: client}, runnerSettings: &ContainerRunnerLocalSettings{Auth: tc.runner}}
			require.NoError(t, runner.pullContainerImage(&ComponentContainerDescription{ContainerConfig: &ContainerConfig{
				Repository: tc.repository, Tag: tc.tag, Auth: tc.image,
			}}))
			require.True(t, pulled.Load())
		})
	}
}

func TestContainerRegistryAuth(t *testing.T) {
	ecr := authSettings{Username: "AWS", Password: "ecr-token", ServerAddress: "registry.example.com"}
	hub := authSettings{Username: "hub-user", Password: "hub-token", ServerAddress: "https://index.docker.io/v1/"}
	upper := authSettings{Username: "user", Password: "token", ServerAddress: "HTTPS://REGISTRY.EXAMPLE.COM/v1/"}
	port := authSettings{Username: "user", Password: "token", ServerAddress: "http://registry.example.com:5000/v1/"}
	for _, tc := range []struct {
		name       string
		repository string
		runner     authSettings
		image      authSettings
		want       docker.AuthConfiguration
	}{
		{name: "private registry", repository: "registry.example.com/mysql", runner: ecr, want: ecr.GetAuthConfig()},
		{name: "public Floci", repository: "floci/floci", runner: ecr},
		{name: "other private registry", repository: "other.example.com/mysql", runner: ecr},
		{name: "Docker Hub alias", repository: "docker.io/floci/floci", runner: hub, want: hub.GetAuthConfig()},
		{name: "implicit Docker Hub", repository: "floci/floci", runner: hub, want: hub.GetAuthConfig()},
		{name: "Docker Hub registry alias", repository: "registry-1.docker.io/floci/floci", runner: hub, want: hub.GetAuthConfig()},
		{name: "unknown fallback registry", repository: "floci/floci", runner: authSettings{Username: "AWS", Password: "ecr-token"}},
		{name: "bare private hostname", repository: "floci/floci", runner: authSettings{Username: "user", Password: "token", ServerAddress: "registry"}},
		{name: "uppercase registry URL", repository: "registry.example.com/mysql", runner: upper, want: upper.GetAuthConfig()},
		{name: "matching registry port", repository: "registry.example.com:5000/mysql", runner: port, want: port.GetAuthConfig()},
		{name: "different registry port", repository: "registry.example.com:5000/mysql", runner: ecr},
		{name: "image auth wins", repository: "registry.example.com/mysql", runner: ecr, image: hub, want: hub.GetAuthConfig()},
		{name: "registry port", repository: "localhost:5000/floci", runner: ecr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := &containerRunnerLocal{runnerSettings: &ContainerRunnerLocalSettings{Auth: tc.runner}}
			assert.Equal(t, tc.want, runner.getAuthConfig(tc.repository, tc.image))
		})
	}
}
