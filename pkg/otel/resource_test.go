package otel_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/otel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

func TestBuildResource(t *testing.T) {
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "operator.attribute=from-env,service.name=from-env")

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

	env, ok := attrs.Value(semconv.DeploymentEnvironmentNameKey)
	require.True(t, ok)
	assert.Equal(t, "production", env.AsString())

	serviceNamespace, ok := attrs.Value(semconv.ServiceNamespaceKey)
	require.True(t, ok)
	assert.Equal(t, "examples", serviceNamespace.AsString())

	org, ok := attrs.Value("organization")
	require.True(t, ok)
	assert.Equal(t, "acme", org.AsString())

	team, ok := attrs.Value("team")
	require.True(t, ok)
	assert.Equal(t, "gosoline", team.AsString())

	operatorAttribute, ok := attrs.Value("operator.attribute")
	require.True(t, ok)
	assert.Equal(t, "from-env", operatorAttribute.AsString())

	telemetrySDKName, ok := attrs.Value(semconv.TelemetrySDKNameKey)
	require.True(t, ok)
	assert.Equal(t, "opentelemetry", telemetrySDKName.AsString())

	telemetrySDKLanguage, ok := attrs.Value(semconv.TelemetrySDKLanguageKey)
	require.True(t, ok)
	assert.Equal(t, "go", telemetrySDKLanguage.AsString())

	telemetrySDKVersion, ok := attrs.Value(semconv.TelemetrySDKVersionKey)
	require.True(t, ok)
	assert.Equal(t, sdk.Version(), telemetrySDKVersion.AsString())

	assert.Equal(t, semconv.SchemaURL, res.SchemaURL())
}

func TestBuildResource_ServiceNamespacePattern(t *testing.T) {
	settings := otel.ResourceSettings{
		ServiceNamePattern: "{app.name}",
		Delimiter:          "-",
	}

	t.Run("missing namespace", func(t *testing.T) {
		config := cfg.New(map[string]any{
			"app": map[string]any{
				"env":  "production",
				"name": "greeting-api",
			},
		})
		settings.ServiceNamespacePattern = "{app.namespace}"

		_, err := otel.BuildResource(config, settings)

		require.EqualError(t, err, "could not format service namespace from pattern \"{app.namespace}\": placeholder {app.namespace} resolved to an empty value in pattern \"{app.namespace}\"")
	})

	t.Run("missing tag", func(t *testing.T) {
		config := cfg.New(map[string]any{
			"app": map[string]any{
				"env":       "production",
				"name":      "greeting-api",
				"namespace": "examples",
			},
		})
		settings.ServiceNamespacePattern = "{app.tags.missing}"

		_, err := otel.BuildResource(config, settings)

		require.EqualError(t, err, "could not format service namespace from pattern \"{app.tags.missing}\": unknown placeholder {app.tags.missing} in pattern \"{app.tags.missing}\"")
	})

	t.Run("empty pattern disables namespace", func(t *testing.T) {
		config := cfg.New(map[string]any{
			"app": map[string]any{
				"env":       "production",
				"name":      "greeting-api",
				"namespace": "examples",
			},
		})
		settings.ServiceNamespacePattern = ""

		res, err := otel.BuildResource(config, settings)
		require.NoError(t, err)

		_, ok := res.Set().Value(semconv.ServiceNamespaceKey)
		assert.False(t, ok)
	})
}
