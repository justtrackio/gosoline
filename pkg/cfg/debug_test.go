package cfg_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/stretchr/testify/require"
)

type debugCredentials struct {
	Opaque      string `cfg:"opaque,secret"`
	AccessKeyID string `cfg:"access_key_id"`
}

func TestDebugConfigUsesOnlySecretAnnotations(t *testing.T) {
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider())
	require.NoError(t, config.Option(cfg.WithConfigSetting("provider", &debugCredentials{
		Opaque: "synthetic-private", AccessKeyID: "public-key",
	}), cfg.WithConfigSetting("credentials_cache.expiry_window", "visible-cache")))
	output, messages := captureDebugConfig(t, config)
	require.NotContains(t, output, "synthetic-private")
	require.Contains(t, messages, "cfg provider.opaque=[REDACTED]")
	require.Contains(t, messages, "cfg provider.access_key_id=public-key")
	require.Contains(t, messages, "cfg credentials_cache.expiry_window=visible-cache")
	require.True(t, config.IsSecret("provider.opaque"))
	require.False(t, config.IsSecret("provider.access_key_id"))
	require.False(t, config.IsSecret("credentials_cache.expiry_window"))
	value, err := config.GetString("provider.opaque")
	require.NoError(t, err)
	require.Equal(t, "synthetic-private", value)
	var decoded debugCredentials
	require.NoError(t, config.UnmarshalKey("provider", &decoded))
	require.Equal(t, "synthetic-private", decoded.Opaque)
}

func TestDebugConfigTypedDynamicContainers(t *testing.T) {
	type namedClients map[string]debugCredentials
	clients := namedClients{"first": {Opaque: "synthetic-first", AccessKeyID: "public-first"}}
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider())
	var schema namedClients
	// Empty typed containers establish the schema for future dynamic entries.
	require.NoError(t, config.Option(cfg.WithConfigSetting("clients", schema)))
	clients["later.with.dot"] = debugCredentials{Opaque: "synthetic-later", AccessKeyID: "public-later"}
	require.NoError(t, config.Option(cfg.WithConfigSetting("clients", clients)))
	list := make([]debugCredentials, 12)
	for i := range list {
		list[i] = debugCredentials{Opaque: fmt.Sprintf("synthetic-%d", i), AccessKeyID: fmt.Sprintf("public-%d", i)}
	}
	require.NoError(t, config.Option(cfg.WithConfigSetting("list", list)))
	output, messages := captureDebugConfig(t, config)
	require.NotContains(t, output, "synthetic-")
	require.Contains(t, output, "public-later")
	for i := range list {
		require.Contains(t, messages, fmt.Sprintf("cfg list.%d.opaque=[REDACTED]", i))
		require.Contains(t, messages, fmt.Sprintf("cfg list.%d.access_key_id=public-%d", i, i))
		require.True(t, config.IsSecret(fmt.Sprintf("list[%d].opaque", i)))
	}
	require.Equal(t, "synthetic-later", clients["later.with.dot"].Opaque)
	require.Equal(t, "synthetic-11", list[11].Opaque)
}

func TestDebugConfigMasksWholeContainerAndFingerprint(t *testing.T) {
	type settings struct {
		Private map[string]string `cfg:"opaque,secret"`
		Public  string            `cfg:"service"`
	}
	fingerprint := func(secret, public string) string {
		config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider())
		require.NoError(t, config.Option(cfg.WithConfigSetting("provider", &settings{
			Private: map[string]string{"synthetic-private-map-key": secret}, Public: public,
		})))
		output, messages := captureDebugConfig(t, config)
		require.NotContains(t, output, secret)
		require.NotContains(t, output, "synthetic-private-map-key")
		require.Contains(t, messages, "cfg provider.opaque=[REDACTED]")
		require.True(t, config.IsSecret("provider.opaque.anything"))
		for _, message := range messages {
			if strings.HasPrefix(message, "cfg fingerprint: ") {
				return message
			}
		}
		t.Fatal("missing fingerprint")

		return ""
	}
	original := fingerprint("synthetic-before", "public")
	require.Equal(t, original, fingerprint("synthetic-after", "public"))
	require.NotEqual(t, original, fingerprint("synthetic-before", "changed-public"))
}

func TestDebugConfigDefaultsSourceAndEnvironmentAnnotations(t *testing.T) {
	type settings struct {
		Credentials   debugCredentials `cfg:"credentials"`
		DefaultSecret string           `cfg:"default_opaque,secret" default:"synthetic-default"`
	}
	env := cfg.NewMemoryEnvProvider()
	require.NoError(t, env.SetEnv("PROVIDER_CREDENTIALS_OPAQUE", "synthetic-env"))
	config := cfg.NewWithInterfaces(env, map[string]any{
		"shared": map[string]any{"opaque": "synthetic-source", "access_key_id": "source-public"},
	})
	require.NoError(t, config.Option(cfg.WithEnvKeyReplacer(cfg.DefaultEnvKeyReplacer)))
	var decoded settings
	require.NoError(t, config.UnmarshalKey("provider", &decoded, cfg.UnmarshalWithDefaultsFromKey("shared", "credentials")))
	require.Equal(t, "synthetic-env", decoded.Credentials.Opaque)
	require.True(t, config.IsSecret("shared.opaque"))
	require.False(t, config.IsSecret("shared.access_key_id"))
	output, messages := captureDebugConfig(t, config)
	require.NotContains(t, output, "synthetic-")
	require.Contains(t, messages, "cfg shared.access_key_id=source-public")
	require.Contains(t, messages, "cfg provider.default_opaque=[REDACTED]")
	outputOnly := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider())
	require.NoError(t, outputOnly.UnmarshalDefaults(&decoded))
	require.False(t, outputOnly.IsSecret("credentials.opaque"))
}

func captureDebugConfig(t *testing.T, config cfg.Config) (outputText string, messages []string) {
	t.Helper()

	var output bytes.Buffer
	handler := log.NewHandlerIoWriter(config, log.PriorityInfo, log.FormatterJson, "main", time.RFC3339, &output)
	logger := log.NewLoggerWithInterfaces(clock.NewFakeClock(), []log.Handler{handler})
	require.NoError(t, cfg.DebugConfig(t.Context(), config, logger))

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	messages = make([]string, len(lines))
	for i, line := range lines {
		var entry struct {
			Message string `json:"message"`
		}
		require.NoError(t, json.Unmarshal([]byte(line), &entry))
		messages[i] = entry.Message
	}

	return output.String(), messages
}
