package cfg_test

import (
	"bytes"
	"context"
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
	output, messages := captureDebugConfig(t, config)
	require.NotContains(t, output, "synthetic-")
	require.Contains(t, messages, "cfg shared.access_key_id=source-public")
	require.Contains(t, messages, "cfg provider.default_opaque=[REDACTED]")
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

func TestDebugConfigIgnoresUnusedConstructorMaps(t *testing.T) {
	stored := map[string]any{
		"client": map[string]any{"opaque": "visible-public-value", "access_key_id": "visible-public-id"},
	}
	ignored := map[string]any{
		"client": debugCredentials{Opaque: "synthetic-ignored-secret", AccessKeyID: "ignored-public-id"},
	}
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), stored, ignored)
	output, messages := captureDebugConfig(t, config)
	require.Contains(t, messages, "cfg client.opaque=visible-public-value")
	require.Contains(t, messages, "cfg client.access_key_id=visible-public-id")
	require.NotContains(t, output, "synthetic-ignored-secret")
	require.Equal(t, stored["client"], config.AllSettings()["client"])
	_, baselineMessages := captureDebugConfig(t, cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), stored))
	require.Equal(t, baselineMessages, messages)
}

type debugCountingLogger struct {
	messages []string
}

func (l *debugCountingLogger) Info(_ context.Context, format string, args ...any) {
	l.messages = append(l.messages, fmt.Sprintf(format, args...))
}

func (l *debugCountingLogger) Error(_ context.Context, _ string, _ ...any) {}

type debugFormattingString string

func (debugFormattingString) String() string {
	panic("configuration logging must not call user formatting")
}

type debugFormattingKey struct{}

func (debugFormattingKey) String() string {
	panic("configuration logging must not format unsupported keys")
}

type debugUnknownConfig struct{ cfg.Config }

func (debugUnknownConfig) AllSettings() map[string]any {
	panic("unknown config must be rejected before reading values")
}

func TestDebugConfigRejectsUnsupportedPublicValuesBeforeLogging(t *testing.T) {
	cyclic := map[string]any{}
	cyclic["self"] = cyclic
	var boxed any = cyclic
	for _, test := range []struct {
		name     string
		value    any
		typeName string
	}{
		{"channel", make(chan string), "chan"},
		{"function", func() {}, "func"},
		{"map_key", map[debugFormattingKey]string{{}: "synthetic-private"}, "debugFormattingKey"},
		{"cycle", &boxed, "map"},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{
				"invalid": test.value, "public": "visible-before-invalid",
			})
			logger := &debugCountingLogger{}
			err := cfg.DebugConfig(t.Context(), config, logger)
			require.Error(t, err)
			require.Contains(t, err.Error(), "invalid")
			require.Contains(t, err.Error(), test.typeName)
			require.NotContains(t, err.Error(), "synthetic-private")
			require.Empty(t, logger.messages)
		})
	}
}

func TestDebugConfigMasksUnsupportedSecretBeforeInspectingIt(t *testing.T) {
	type settings struct {
		Hidden   any                   `cfg:"hidden,secret"`
		Public   debugFormattingString `cfg:"public"`
		Duration time.Duration         `cfg:"duration"`
	}
	cyclic := map[string]any{}
	cyclic["synthetic-private-key"] = cyclic
	var value any = settings{Hidden: cyclic, Public: "visible-scalar", Duration: time.Second}
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{"client": &value})
	output, messages := captureDebugConfig(t, config)
	require.NotContains(t, output, "synthetic-private-key")
	require.Contains(t, messages, "cfg client.hidden=[REDACTED]")
	require.Contains(t, messages, "cfg client.public=visible-scalar")
	require.Contains(t, messages, "cfg client.duration=1s")
	require.Equal(t, cyclic, value.(settings).Hidden)
}

func TestDebugConfigRejectsUnknownConfigBeforeReadingOrLogging(t *testing.T) {
	logger := &debugCountingLogger{}
	err := cfg.DebugConfig(t.Context(), debugUnknownConfig{
		Config: cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider()),
	}, logger)
	require.ErrorContains(t, err, "safe logging metadata")
	require.Empty(t, logger.messages)
}
