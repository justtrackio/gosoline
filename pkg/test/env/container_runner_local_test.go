package env

import (
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
	for _, tag := range []string{"2.3.0", "2.3.0@" + digest} {
		t.Run(tag, func(t *testing.T) {
			var pulled atomic.Bool
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					http.Error(w, "image not found", http.StatusNotFound)

					return
				}
				pulled.Store(true)
				assert.Equal(t, "/images/create", r.URL.Path)
				assert.Equal(t, "floci/floci", r.URL.Query().Get("fromImage"))
				expectedTag := tag
				if tag != "2.3.0" {
					expectedTag = digest
				}
				assert.Equal(t, expectedTag, r.URL.Query().Get("tag"))
				_, err := fmt.Fprintln(w, `{"status":"pulled"}`)
				assert.NoError(t, err)
			}))
			t.Cleanup(server.Close)
			client, err := docker.NewClient(server.URL)
			require.NoError(t, err)
			runner := &containerRunnerLocal{pool: &dockertest.Pool{Client: client}, runnerSettings: &ContainerRunnerLocalSettings{}}
			require.NoError(t, runner.pullContainerImage(&ComponentContainerDescription{ContainerConfig: &ContainerConfig{
				Repository: "floci/floci", Tag: tag,
			}}))
			require.True(t, pulled.Load())
		})
	}
}
