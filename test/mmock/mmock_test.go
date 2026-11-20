//go:build integration && fixtures

package mmock_test

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justtrackio/gosoline/pkg/test/env"
	"github.com/stretchr/testify/require"
)

func TestMmockFixturesAndCapturedRequests(t *testing.T) {
	environment, err := env.NewEnvironment(t,
		env.WithLoggerLevel("error"),
		env.WithConfigMap(map[string]any{
			"app.env": "test", "app.name": "mmock", "app.tags.project": "justtrack", "app.tags.family": "gosoline", "app.tags.group": "test",
			"test.auto_detect.enabled":            false,
			"test.components.mmock.default.mocks": []string{"testdata/events.json", "testdata/status.json"},
		}))
	if environment != nil {
		t.Cleanup(func() { require.NoError(t, environment.Stop()) })
	}
	require.NoError(t, err)
	component := environment.Mmock("default")
	body := `{"items":["one","two"],"event":"install"}`
	code, response := call(t, http.MethodPost, component.Address()+"/events?token=test", body)
	require.Equal(t, http.StatusOK, code)
	require.JSONEq(t, `{"accepted":true}`, response)
	requests, err := component.Requests(t.Context())
	require.NoError(t, err)
	require.Len(t, requests, 1)
	require.Equal(t, http.MethodPost, requests[0].Method)
	require.Equal(t, "/events", requests[0].Path)
	require.Equal(t, []string{"test"}, requests[0].QueryStringParameters["token"])
	require.Equal(t, []string{"application/json"}, requests[0].Headers["Content-Type"])
	require.JSONEq(t, `{"event":"install","items":["one","two"]}`, requests[0].Body)

	require.NoError(t, component.ResetRequests(t.Context()))
	requests, err = component.Requests(t.Context())
	require.NoError(t, err)
	require.Empty(t, requests)
	code, response = call(t, http.MethodPost, component.Address()+"/events?token=test", body)
	require.Equal(t, http.StatusOK, code)
	require.JSONEq(t, `{"accepted":true}`, response)
	code, _ = call(t, http.MethodPost, component.Address()+"/events?token=wrong", body)
	require.Equal(t, http.StatusNotFound, code)
	code, _ = call(t, http.MethodPost, component.Address()+"/events?token=test", `{"event":"purchase"}`)
	require.Equal(t, http.StatusNotFound, code)
	code, response = call(t, http.MethodGet, component.Address()+"/status", "")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "ready", response)
	code, response = call(t, http.MethodGet, component.AdminAddress(), "")
	require.Equal(t, http.StatusOK, code)
	require.Contains(t, response, "<html")
}

func call(t *testing.T, method, address, body string) (statusCode int, responseBody string) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), method, address, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	data, err := io.ReadAll(response.Body)
	require.NoError(t, response.Body.Close())
	require.NoError(t, err)

	return response.StatusCode, string(data)
}

func TestMmockReportsFixtureErrors(t *testing.T) {
	for _, testCase := range []struct{ name, contents, want string }{
		{name: "missing", want: "could not read mmock fixture"},
		{name: "invalid", contents: `{"URI":"invalid.json","request":{}}`, want: "400 Bad Request"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			filename := filepath.Join(t.TempDir(), "fixture.json")
			if testCase.contents != "" {
				require.NoError(t, os.WriteFile(filename, []byte(testCase.contents), 0o600))
			}
			environment, err := env.NewEnvironment(t, env.WithLoggerLevel("error"), env.WithConfigMap(map[string]any{
				"app.env": "test", "app.name": "mmock", "app.tags.project": "justtrack", "app.tags.family": "gosoline", "app.tags.group": "test",
				"test.auto_detect.enabled": false, "test.components.mmock.default.mocks": []string{filename},
			}))
			if environment != nil {
				t.Cleanup(func() { require.NoError(t, environment.Stop()) })
			}
			require.ErrorContains(t, err, testCase.want)
			require.ErrorContains(t, err, filename)
		})
	}
}
