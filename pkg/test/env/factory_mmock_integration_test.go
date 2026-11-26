//go:build integration && fixtures

package env

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMmockBinaryResponseFile(t *testing.T) {
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	_, err := writer.Write([]byte("country,revenue\nDE,1.25\n"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	directory := t.TempDir()
	asset := filepath.Join(directory, "response.csv.gz")
	require.NoError(t, os.WriteFile(asset, compressed.Bytes(), 0o600))
	fixture := filepath.Join(directory, "response.yaml")
	require.NoError(t, os.WriteFile(fixture, []byte(`URI: response.yaml
request:
  method: GET
  path: /report
response:
  statusCode: 200
  headers:
    Content-Type: [application/gzip]
    Content-Encoding: [gzip]
  body: '{{file.contents(/tmp/mmock/response.csv.gz)}}'
`), 0o600))
	environment, err := NewEnvironment(t, WithConfigMap(map[string]any{
		"app.env":                             "test",
		"app.name":                            "mmock-file-copy",
		"test.container_manager.runner_type":  RunnerTypeLocal,
		"test.components.mmock.default.files": []ContainerFile{{Source: asset, Target: "/tmp/mmock/response.csv.gz"}},
		"test.components.mmock.default.mocks": []string{fixture},
	}))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, environment.Stop()) })
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, environment.Mmock("default").Address()+"/report", http.NoBody)
	require.NoError(t, err)
	// Keep the compressed wire bytes so this checks the file transfer itself.
	request.Header.Set("Accept-Encoding", "gzip")
	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer func() { require.NoError(t, response.Body.Close()) }()
	data, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, compressed.Bytes(), data)
}

func TestMmockLoadedFixturesRemainAvailable(t *testing.T) {
	for run := range 10 {
		t.Run(fmt.Sprint(run), func(t *testing.T) {
			directory := t.TempDir()
			var fixtures []string
			for i := range 25 {
				filename := filepath.Join(directory, fmt.Sprintf("%02d.json", i))
				require.NoError(t, os.WriteFile(filename, fmt.Appendf(nil,
					`{"URI":"fixture-%[1]d.json","request":{"method":"GET","path":"/fixture/%[1]d"},"response":{"statusCode":200}}`, i), 0o600))
				fixtures = append(fixtures, filename)
			}

			environment, err := NewEnvironment(t, WithConfigMap(map[string]any{
				"app.env":                             "test",
				"app.name":                            "mmock-loaded-fixtures",
				"test.container_manager.runner_type":  RunnerTypeLocal,
				"test.components.mmock.default.mocks": fixtures,
			}))
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, environment.Stop()) })
			client := &http.Client{Timeout: time.Second}
			// Exercise requests across the first file-watcher tick after registration.
			for deadline := time.Now().Add(250 * time.Millisecond); time.Now().Before(deadline); {
				request, err := http.NewRequestWithContext(t.Context(), http.MethodGet,
					environment.Mmock("default").Address()+"/fixture/24", http.NoBody)
				require.NoError(t, err)
				response, err := client.Do(request)
				require.NoError(t, err)
				require.NoError(t, response.Body.Close())
				require.Equal(t, http.StatusOK, response.StatusCode)
			}
		})
	}
}
