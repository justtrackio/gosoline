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

func TestDebugConfigMasksSensitiveValues(t *testing.T) {
	for _, keyword := range []string{"PaSsWoRd", "SeCrEt", "ToKeN", "KeY", "DsN", "CrEdEnTiAl"} {
		t.Run(keyword, func(t *testing.T) {
			key := "prefix" + keyword + "suffix"
			secret := "synthetic-private-" + keyword
			settings := map[string]any{
				key:                secret,
				"nested":           map[string]any{key: secret},
				"parent" + keyword: map[string]any{"value": secret},
				"service":          "payment-service",
				"port":             8080,
			}
			config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), settings)

			output, messages := captureDebugConfig(t, config)

			require.NotContains(t, output, secret)
			require.Contains(t, output, "cfg "+key+"=")
			require.Contains(t, output, "cfg nested."+key+"=")
			require.Contains(t, output, "cfg parent"+keyword+".value=")
			require.Contains(t, messages, "cfg service=payment-service")
			require.Contains(t, messages, "cfg port=8080")

			for _, path := range []string{key, "nested." + key, "parent" + keyword + ".value"} {
				value, err := config.GetString(path)
				require.NoError(t, err)
				require.Equal(t, secret, value)
			}
		})
	}
}

func TestDebugConfigMasksEveryListPosition(t *testing.T) {
	clients := make([]any, 12)
	keys := make([]any, len(clients))
	for i := range clients {
		clients[i] = map[string]any{
			"auth": map[string]any{"accessToKeN": fmt.Sprintf("synthetic-client-private-%d", i)},
			"name": fmt.Sprintf("visible-client-%d", i),
		}
		keys[i] = fmt.Sprintf("synthetic-ring-private-%d", i)
	}
	settings := map[string]any{"clients": clients, "KeYRing": keys}
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), settings)

	output, messages := captureDebugConfig(t, config)

	for i := range clients {
		require.NotContains(t, output, fmt.Sprintf("synthetic-client-private-%d", i))
		require.NotContains(t, output, fmt.Sprintf("synthetic-ring-private-%d", i))
		require.Contains(t, output, fmt.Sprintf("cfg clients.%d.auth.accessToKeN=", i))
		require.Contains(t, messages, fmt.Sprintf("cfg clients.%d.name=visible-client-%d", i, i))
		require.Contains(t, output, fmt.Sprintf("cfg KeYRing.%d=", i))
	}
	storedClients, err := config.GetMsiSlice("clients")
	require.NoError(t, err)
	require.Len(t, storedClients, len(clients))
	for i, client := range storedClients {
		require.Equal(t, map[string]any{
			"auth": map[string]any{"accessToKeN": fmt.Sprintf("synthetic-client-private-%d", i)},
			"name": fmt.Sprintf("visible-client-%d", i),
		}, client)
	}
	storedKeys, err := config.GetStringSlice("KeYRing")
	require.NoError(t, err)
	for i := range storedKeys {
		require.Equal(t, fmt.Sprintf("synthetic-ring-private-%d", i), storedKeys[i])
	}
	require.Len(t, storedKeys, len(keys))
}

func TestDebugConfigFingerprintUsesRedactedValues(t *testing.T) {
	for _, keyword := range []string{"PaSsWoRd", "SeCrEt", "ToKeN", "KeY", "DsN", "CrEdEnTiAl"} {
		t.Run(keyword, func(t *testing.T) {
			fingerprint := func(secret, service string) string {
				config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{
					"clients": []any{
						map[string]any{keyword: secret},
						map[string]any{keyword: secret},
						map[string]any{keyword: secret},
					},
					"service": service,
				})
				_, messages := captureDebugConfig(t, config)
				for _, message := range messages {
					if strings.HasPrefix(message, "cfg fingerprint: ") {
						return message
					}
				}
				t.Fatal("missing config fingerprint")
				return ""
			}

			original := fingerprint("synthetic-private-before", "payment-service")
			require.Equal(t, original, fingerprint("synthetic-private-after", "payment-service"))
			require.NotEqual(t, original, fingerprint("synthetic-private-before", "changed-service"))
		})
	}
}

func captureDebugConfig(t *testing.T, config cfg.Config) (string, []string) {
	t.Helper()

	var output bytes.Buffer
	handler := log.NewHandlerIoWriter(config, log.PriorityInfo, log.FormatterJson, "main", time.RFC3339, &output)
	logger := log.NewLoggerWithInterfaces(clock.NewFakeClock(), []log.Handler{handler})
	require.NoError(t, cfg.DebugConfig(t.Context(), config, logger))

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	messages := make([]string, len(lines))
	for i, line := range lines {
		var entry struct {
			Message string `json:"message"`
		}
		require.NoError(t, json.Unmarshal([]byte(line), &entry))
		messages[i] = entry.Message
	}

	return output.String(), messages
}
