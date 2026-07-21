package log

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlerOtelFactory_ProvidesShutdown(t *testing.T) {
	ctx := WithShutdownContainer(t.Context())

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

	_, err = handlerOtelFactory(ctx, config, "otel")
	require.NoError(t, err)

	// Verify shutdown was stored in container
	c, ok := ctx.Value(logShutdownKey{}).(*shutdownContainer)
	require.True(t, ok)
	assert.NotNil(t, c.fn)
}
