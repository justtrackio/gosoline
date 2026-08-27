package metric_test

import (
	"context"
	"strings"
	"testing"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaVersionConstant(t *testing.T) {
	assert.Equal(t, "v1.0", metric.SchemaVersion)
	assert.Equal(t, metric.SchemaVersion, strings.TrimSpace(metric.SchemaVersion))
	assert.True(t, metric.IsValidSchemaVersion(metric.SchemaVersion))
}

func TestIsValidSchemaVersion(t *testing.T) {
	cases := map[string]bool{
		"v0.0":          true,
		"v1.0":          true,
		"v10.3":         true,
		"v999999999.0":  true,
		"":              false,
		"1.0":           false,
		"v1":            false,
		"v1.0.0":        false,
		"v01.0":         false,
		"v1.00":         false,
		" v1.0":         false,
		"v1.0 ":         false,
		"vx.y":          false,
		"v1234567890.0": false,
	}

	for version, expected := range cases {
		assert.Equal(t, expected, metric.IsValidSchemaVersion(version), "version %q", version)
	}
}

func TestRegisterSchemaVersion(t *testing.T) {
	ctx := appctx.WithContainer(context.Background())

	require.NoError(t, metric.RegisterSchemaVersion(ctx))

	metadata, err := appctx.ProvideMetadata(ctx)
	require.NoError(t, err, "the metadata carrier should be available")

	version, ok := metadata.Get(metric.MetadataKeySchemaVersion).Data().(string)
	require.True(t, ok, "the schema version should be stored as a string")
	assert.Equal(t, metric.SchemaVersion, version, "the registered value must be byte identical to the constant")

	assert.Equal(t, map[string]any{
		"metric": map[string]any{
			"schema_version": metric.SchemaVersion,
		},
	}, metadata.Msi(), "the dot notation key should nest as metric -> schema_version")
}

func TestRegisterSchemaVersionKeepsForeignEntries(t *testing.T) {
	ctx := appctx.WithContainer(context.Background())

	metadata, err := appctx.ProvideMetadata(ctx)
	require.NoError(t, err, "the metadata carrier should be available")

	producers := []any{map[string]any{"name": "producer-1"}}
	queues := []any{map[string]any{"queueName": "queue-1"}}

	metadata.Set("stream.producers", producers)
	metadata.Set("cloud.aws.sqs.queues", queues)

	leavesBefore := countLeaves(metadata.Msi())

	require.NoError(t, metric.RegisterSchemaVersion(ctx))

	msi := metadata.Msi()

	assert.Equal(t, producers, msi["stream"].(map[string]any)["producers"], "foreign stream entries should stay unchanged")
	assert.Equal(t, queues, msi["cloud"].(map[string]any)["aws"].(map[string]any)["sqs"].(map[string]any)["queues"], "foreign sqs entries should stay unchanged")
	assert.Equal(t, leavesBefore+1, countLeaves(msi), "registration should add exactly one leaf")
}

func TestRegisterSchemaVersionWithoutContainer(t *testing.T) {
	ctx := &countingContext{Context: context.Background()}

	err := metric.RegisterSchemaVersion(ctx)

	require.Error(t, err, "registration without an appctx container should fail")
	assert.Contains(t, err.Error(), "metric schema version", "the error should name the failing operation")

	// appctx.Provide resolves the container through a single ctx.Value lookup, so one lookup proves
	// registration performed exactly one attempt without any retry.
	assert.Equal(t, 1, ctx.lookups, "registration should attempt exactly once")

	var causeNoContainer *appctx.ErrNoApplicationContainerFound
	assert.ErrorAs(t, err, &causeNoContainer, "the underlying cause should stay unwrappable")
	assert.Contains(t, err.Error(), causeNoContainer.Error(), "the cause message should be wrapped unchanged")
}

// countingContext counts the ctx.Value lookups performed against it.
type countingContext struct {
	context.Context
	lookups int
}

func (c *countingContext) Value(key any) any {
	c.lookups++

	return c.Context.Value(key)
}

// countLeaves counts the non map values of a nested metadata document.
func countLeaves(values map[string]any) int {
	var leaves int

	for _, value := range values {
		if nested, ok := value.(map[string]any); ok {
			leaves += countLeaves(nested)

			continue
		}

		leaves++
	}

	return leaves
}
