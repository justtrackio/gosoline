package otel_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/otel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func TestBuildResource(t *testing.T) {
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"env":       "production",
			"name":      "greeting-api",
			"namespace": "examples",
			"tags": map[string]any{
				"project": "gosoline",
			},
		},
	})

	settings := otel.ResourceSettings{
		ServiceNamePattern:      "{app.name}",
		ServiceNamespacePattern: "{app.namespace}",
		Delimiter:               "-",
		Attributes: map[string]string{
			"organization": "acme",
			"team":         "{app.tags.project}",
		},
	}

	res, err := otel.BuildResource(config, settings)
	require.NoError(t, err)

	attrs := res.Set()

	serviceName, ok := attrs.Value(semconv.ServiceNameKey)
	require.True(t, ok)
	assert.Equal(t, "greeting-api", serviceName.AsString())

	env, ok := attrs.Value(semconv.DeploymentEnvironmentKey)
	require.True(t, ok)
	assert.Equal(t, "production", env.AsString())

	org, ok := attrs.Value("organization")
	require.True(t, ok)
	assert.Equal(t, "acme", org.AsString())

	team, ok := attrs.Value("team")
	require.True(t, ok)
	assert.Equal(t, "gosoline", team.AsString())
}
