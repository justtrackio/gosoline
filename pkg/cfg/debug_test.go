package cfg_test

import (
	"bytes"
	"crypto/md5"
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
				key:       secret,
				"nested":  map[string]any{key: secret},
				"service": "payment-service",
				"port":    8080,
			}
			config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), settings)

			output, messages := captureDebugConfig(t, config)

			require.NotContains(t, output, secret)
			require.Contains(t, output, "cfg "+key+"=")
			require.Contains(t, output, "cfg nested."+key+"=")
			require.Contains(t, messages, "cfg service=payment-service")
			require.Contains(t, messages, "cfg port=8080")

			for _, path := range []string{key, "nested." + key} {
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

func captureDebugConfig(t *testing.T, config cfg.Config) (outputText string, messages []string) {
	t.Helper()

	var output bytes.Buffer
	handler := log.NewHandlerIoWriter(config, log.PriorityInfo, log.FormatterJson, "main", time.RFC3339, &output)
	logger := log.NewLoggerWithInterfaces(clock.NewFakeClock(), []log.Handler{handler})
	// Initialize the handler's lazy channel settings before testing DebugConfig purity.
	logger.Info(t.Context(), "fixture logger warmup")
	output.Reset()
	settingsBeforeDebug := config.AllSettings()
	require.NoError(t, cfg.DebugConfig(t.Context(), config, logger))
	require.Equal(t, settingsBeforeDebug, config.AllSettings())

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	messages = make([]string, len(lines))
	for i, line := range lines {
		var entry struct {
			Message string `json:"message"`
		}
		require.NoError(t, json.Unmarshal([]byte(line), &entry))
		messages[i] = entry.Message
	}

	entries := make([]string, len(messages)-1)
	for i, message := range messages[:len(messages)-1] {
		entries[i] = strings.TrimPrefix(message, "cfg ")
	}
	require.Equal(t, fmt.Sprintf("cfg fingerprint: %x", md5.Sum([]byte(strings.Join(entries, ";")))), messages[len(messages)-1])

	return output.String(), messages
}

func TestDebugConfigEntropyBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name   string
		value  string
		masked bool
	}{
		{"empty", "", false},
		{"11_bytes", "ABCDEFGHIJK", false},
		{"12_bytes", "ABCDEFGHIJKL", true},
		{"12_repeated", "ABCDEFABCDEF", false},
		{"20_exact_adaptive_boundary", "ABCDEFGHIJKLMNOPQRSA", true},
		{"20_below_adaptive_boundary", "ABCDEFGHIJKLMNOPQRAB", false},
		{"24_above_adaptive_boundary", "ABCDEFGHIJKLMNOPQRSTUVWA", true},
		{"24_below_adaptive_boundary", "ABCDEFGHIJKLMNOPQRSTUVAB", false},
		{"25_above_capped_boundary", "ABCDEFGHIJKLMNOPQRSTUVWXA", true},
		{"25_below_capped_boundary", "ABCDEFGHIJKLMNOPQRSTUVWAB", false},
		{"31_bytes", "ABCDEFGHIJKLMNOPQRSTUVWXYZabcde", true},
		{"32_bytes", "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdef", true},
		{"exact_4_5", "ABCDEFGHABCDEFGHijklmnopqrstuvwx", true},
		{"below_4_5", "ABCDEFGHIJABCDEFGHIJklmnopqrstuv", false},
		{"22_symbols", strings.Repeat("ABCDEFGHIJKLMNOPQRSTUV", 2), false},
		{"23_symbols", strings.Repeat("ABCDEFGHIJKLMNOPQRSTUVW", 2), true},
		{"repeated", strings.Repeat("a", 128), false},
		{"uuid", "79db3180-99a9-4157-91c3-a591b9a8f01c", false},
		{"hex", strings.Repeat("0123456789abcdef", 4), false},
		{"public_url", "https://api.example.com/v1/metrics", false},
		{"short_public_url", "https://x.io", false},
		{"public_base64_false_positive", "c29tZSBzdHJpbmcgd2l0aCDDvG5pY8O4ZMOr", true},
		{"unicode_repeated_bytes", strings.Repeat("é", 32), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{"opaque": tc.value})
			output, messages := captureDebugConfig(t, config)
			if tc.masked {
				require.NotContains(t, output, tc.value)
				require.Contains(t, messages, "cfg opaque=[REDACTED]")
			} else {
				require.Contains(t, messages, "cfg opaque="+tc.value)
			}
			stored, err := config.GetString("opaque")
			require.NoError(t, err)
			require.Equal(t, tc.value, stored)
		})
	}
}

func TestDebugConfigEntropyEmbeddedValues(t *testing.T) {
	private := "ABCDEFGHIJKL"
	padding := strings.Repeat("x", 512)
	for _, value := range []string{
		"Bearer " + private,
		"postgres://user:" + private + "@localhost/db?application_name=" + padding,
		"host=localhost password=" + private + " application_name=" + padding,
		"https://localhost/?auth=" + private + "&public=" + padding,
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9." + padding + "." + private,
		padding + "=" + private + "==",
	} {
		config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{"endpoint": value})
		output, messages := captureDebugConfig(t, config)
		require.NotContains(t, output, private)
		require.Contains(t, messages, "cfg endpoint=[REDACTED]")
		require.Equal(t, value, config.AllSettings()["endpoint"])
	}
}

func TestDebugConfigEntropyUsesFormattedLeaves(t *testing.T) {
	type namedString string
	type namedMap map[string]string
	private := "ABCDEFGHIJKL"
	for _, value := range []any{
		namedString(private),
		[]string{"visible", private},
		namedMap{"opaque": private},
		struct{ Opaque string }{private},
		&struct{ Opaque string }{private},
	} {
		config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{"opaque": value})
		output, messages := captureDebugConfig(t, config)
		require.NotContains(t, output, private)
		require.Contains(t, messages, "cfg opaque=[REDACTED]")
		require.Equal(t, value, config.AllSettings()["opaque"])
	}
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{
		"number": 8080, "enabled": true, "duration": time.Second,
		"bytes": []byte{1, 2, 3},
	})
	_, messages := captureDebugConfig(t, config)
	require.Contains(t, messages, "cfg number=8080")
	require.Contains(t, messages, "cfg enabled=true")
	require.Contains(t, messages, "cfg duration=1s")
	require.Contains(t, messages, "cfg bytes=[1 2 3]")
}

func TestDebugConfigEntropyMasksEveryListPosition(t *testing.T) {
	private := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdef"
	clients := make([]any, 12)
	for i := range clients {
		clients[i] = map[string]any{"opaque": private, "name": fmt.Sprintf("visible-%d", i)}
	}
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{"clients": clients})
	output, messages := captureDebugConfig(t, config)
	require.NotContains(t, output, private)
	for i := range clients {
		require.Contains(t, messages, fmt.Sprintf("cfg clients.%d.opaque=[REDACTED]", i))
		require.Contains(t, messages, fmt.Sprintf("cfg clients.%d.name=visible-%d", i, i))
	}
	require.Equal(t, clients, config.AllSettings()["clients"])
}

func TestDebugConfigRecognizesExistingCredentialKeys(t *testing.T) {
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{
		"httpserver":         map[string]any{"api_auth_basic_users": []any{"alice:old", "bob:new"}},
		"headers":            map[string]any{"aUtHoRiZaTiOn": []any{"Basic YTpi"}, "other": "visible"},
		"authorization_mode": "visible", "not_api_auth_basic_users": "visible",
		"monkey": "short", "keyboard": "short",
	})
	output, messages := captureDebugConfig(t, config)
	require.NotContains(t, output, "alice:old")
	require.NotContains(t, output, "bob:new")
	require.NotContains(t, output, "Basic YTpi")
	require.Contains(t, messages, "cfg httpserver.api_auth_basic_users.0=[REDACTED]")
	require.Contains(t, messages, "cfg httpserver.api_auth_basic_users.1=[REDACTED]")
	require.Contains(t, messages, "cfg headers.aUtHoRiZaTiOn.0=[REDACTED]")
	require.Contains(t, messages, "cfg headers.other=visible")
	require.Contains(t, messages, "cfg authorization_mode=visible")
	require.Contains(t, messages, "cfg not_api_auth_basic_users=visible")
	require.Contains(t, messages, "cfg monkey=[REDACTED]")
	require.Contains(t, messages, "cfg keyboard=[REDACTED]")
}

func TestDebugConfigEntropyFingerprintAndStorage(t *testing.T) {
	fingerprint := func(private, public string) string {
		config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{
			"opaque": private, "service": public,
		})
		_, messages := captureDebugConfig(t, config)
		require.Equal(t, private, config.AllSettings()["opaque"])
		var decoded struct {
			Opaque string `cfg:"opaque"`
		}
		require.NoError(t, config.UnmarshalKey(".", &decoded))
		require.Equal(t, private, decoded.Opaque)
		for _, message := range messages {
			if strings.HasPrefix(message, "cfg fingerprint: ") {
				return message
			}
		}
		t.Fatal("missing config fingerprint")

		return ""
	}
	original := fingerprint("ABCDEFGHIJKL", "visible")
	require.Equal(t, original, fingerprint("abcdefghijkl", "visible"))
	require.NotEqual(t, original, fingerprint("ABCDEFGHIJKL", "changed"))
}

func TestDebugConfigEntropyDoesNotIncludeKey(t *testing.T) {
	name := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdef"
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{name: "short"})
	_, messages := captureDebugConfig(t, config)
	require.Contains(t, messages, "cfg "+name+"=short")
}

func TestDebugConfigPublicParentKeysRemainVisible(t *testing.T) {
	for _, keyword := range []string{"PaSsWoRd", "SeCrEt", "ToKeN", "KeY", "DsN", "CrEdEnTiAl"} {
		t.Run(keyword, func(t *testing.T) {
			key := "parent" + keyword
			fingerprint := func(value string) string {
				settings := map[string]any{key: map[string]any{"expiry_window": value}}
				config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), settings)
				_, messages := captureDebugConfig(t, config)
				require.Contains(t, messages, "cfg "+key+".expiry_window="+value)
				stored, err := config.GetString(key + ".expiry_window")
				require.NoError(t, err)
				require.Equal(t, value, stored)

				return messages[len(messages)-1]
			}
			require.NotEqual(t, fingerprint("5m"), fingerprint("10m"))
		})
	}
}

func TestDebugConfigCredentialsCacheExpiryRemainsVisible(t *testing.T) {
	settings := map[string]any{
		"cloud": map[string]any{
			"aws": map[string]any{
				"s3": map[string]any{
					"clients": map[string]any{
						"default": map[string]any{
							"credentials_cache": map[string]any{"expiry_window": "5m"},
						},
					},
				},
			},
		},
	}
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), settings)
	_, messages := captureDebugConfig(t, config)
	require.Contains(t, messages, "cfg cloud.aws.s3.clients.default.credentials_cache.expiry_window=5m")
	stored, err := config.GetDuration("cloud.aws.s3.clients.default.credentials_cache.expiry_window")
	require.NoError(t, err)
	require.Equal(t, 5*time.Minute, stored)
}

func TestDebugConfigNestedListOwnersAndCredentialContainers(t *testing.T) {
	rows := make([]any, 12)
	for i := range rows {
		rows[i] = []any{"old", "new"}
	}
	settings := map[string]any{
		"KeYRing":            rows,
		"native_keyring":     []string{"old", "new"},
		"public_token_cache": map[string]any{"durations": rows},
		"nested": []any{
			map[string]any{
				"headers":    map[string]any{"aUtHoRiZaTiOn": map[string]any{"values": rows}},
				"httpserver": map[string]any{"api_auth_basic_users": rows},
			},
		},
	}
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), settings)
	output, messages := captureDebugConfig(t, config)
	require.NotContains(t, output, "cfg native_keyring=[old new]")
	require.Contains(t, messages, "cfg native_keyring=[REDACTED]")
	for i := range rows {
		for j, value := range []string{"old", "new"} {
			require.Contains(t, messages, fmt.Sprintf("cfg KeYRing.%d.%d=[REDACTED]", i, j))
			require.Contains(t, messages, fmt.Sprintf("cfg public_token_cache.durations.%d.%d=%s", i, j, value))
			require.Contains(t, messages, fmt.Sprintf("cfg nested.0.headers.aUtHoRiZaTiOn.values.%d.%d=[REDACTED]", i, j))
			require.Contains(t, messages, fmt.Sprintf("cfg nested.0.httpserver.api_auth_basic_users.%d.%d=[REDACTED]", i, j))
		}
	}
	native, err := config.GetStringSlice("native_keyring")
	require.NoError(t, err)
	require.Equal(t, []string{"old", "new"}, native)
}

func TestDebugConfigShortEmbeddedEntropyBoundaries(t *testing.T) {
	for _, tc := range []struct {
		token  string
		masked bool
	}{
		{"ABCDEFGHIJK", false},
		{"ABCDEFGHIJKL", true},
		{"ABCDEFABCDEF", false},
	} {
		t.Run(tc.token, func(t *testing.T) {
			value := strings.Repeat("x", 512) + "=" + tc.token + "=="
			config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{"opaque": value})
			output, messages := captureDebugConfig(t, config)
			if tc.masked {
				require.NotContains(t, output, tc.token)
				require.Contains(t, messages, "cfg opaque=[REDACTED]")
			} else {
				require.Contains(t, messages, "cfg opaque="+value)
			}
			stored, err := config.GetString("opaque")
			require.NoError(t, err)
			require.Equal(t, value, stored)
		})
	}
}
