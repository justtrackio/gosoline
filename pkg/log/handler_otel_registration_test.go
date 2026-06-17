package log

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlerOtelFactory_RegistersShutdown(t *testing.T) {
	ResetShutdownRegistry()
	t.Cleanup(ResetShutdownRegistry)

	config := cfg.New()
	err := config.Option(cfg.WithConfigMap(map[string]any{
		"app": map[string]any{
			"name": "test",
			"env":  "test",
		},
		"log": map[string]any{
			"handlers": map[string]any{
				"otel": map[string]any{
					"type":  "otel",
					"level": "info",
				},
			},
		},
	}))
	require.NoError(t, err)

	require.Empty(t, shutdownEntries, "registry should start empty")

	_, err = handlerOtelFactory(config, "otel")
	require.NoError(t, err)

	require.Len(t, shutdownEntries, 1, "otel handler must register exactly one shutdown entry")
	assert.Equal(t, "otel", shutdownEntries[0].name)
}
