package env

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMmockFactoryLoadsFixtures(t *testing.T) {
	for _, format := range []string{"json", "yaml", "yml"} {
		t.Run(format, func(t *testing.T) {
			content := `{"request":{"method":"POST","path":"/example"},"response":{"statusCode":200,"body":"{\n  \"accepted\": true\n}"}}`
			if format != "json" {
				content = "request:\n  method: POST\n  path: /example\nresponse:\n  statusCode: 200\n  body: |-\n    {\n      \"accepted\": true\n    }\n"
			}
			filename := filepath.Join(t.TempDir(), "mock."+format)
			require.NoError(t, os.WriteFile(filename, []byte(content), 0o600))
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/api/mapping/0-mock."+format, r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				assert.JSONEq(t, `{"request":{"method":"POST","path":"/example"},"response":{"statusCode":200,"body":"{\n  \"accepted\": true\n}"}}`, string(body))
				w.WriteHeader(http.StatusCreated)
			}))
			defer server.Close()
			host, port, err := net.SplitHostPort(strings.TrimPrefix(server.URL, "http://"))
			require.NoError(t, err)
			containers := map[string]*Container{"main": {bindings: map[string]ContainerBinding{
				"admin": {host: host, port: port},
				"main":  {host: "localhost", port: "8083"},
			}}}
			component, err := new(mmockFactory).Component(nil, nil, containers, &mmockSettings{Mocks: []string{filename}})
			require.NoError(t, err)
			assert.Equal(t, "http://localhost:8083", component.(*MmockComponent).Address())
			assert.Equal(t, server.URL, component.(*MmockComponent).AdminAddress())
		})
	}
}

func TestMmockFactoryRejectsInvalidYAML(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "invalid.yaml")
	require.NoError(t, os.WriteFile(filename, []byte("request: [\n"), 0o600))
	_, err := new(mmockFactory).Component(nil, nil, map[string]*Container{"main": {}}, &mmockSettings{Mocks: []string{filename}})
	require.ErrorContains(t, err, "could not parse mmock fixture")
	require.ErrorContains(t, err, filename)
}
