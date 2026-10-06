package log

import (
	"testing"

	"github.com/stretchr/testify/assert"
	otellog "go.opentelemetry.io/otel/log"
)

func TestAppendOtelAttributesPreservesContextAndConflictingFields(t *testing.T) {
	attributes := make([]otellog.KeyValue, 0, 5)
	keys := make(map[string]struct{}, 5)

	attributes = appendOtelAttribute(attributes, keys, "channel", "metadata", "sdkLogs")
	attributes = appendOtelAttributes(attributes, keys, "context", map[string]any{
		"sdk_platform": "android",
		"sdk_version":  "8.0.0",
	})
	attributes = appendOtelAttributes(attributes, keys, "fields", map[string]any{
		"level":       "info",
		"sdk_version": "13.2.0",
	})

	got := make(map[string]string, len(attributes))
	for _, attribute := range attributes {
		if assert.NotContains(t, got, attribute.Key) {
			got[attribute.Key] = attribute.Value.AsString()
		}
	}

	assert.Len(t, got, len(attributes))
	assert.Equal(t, map[string]string{
		"channel":            "sdkLogs",
		"fields.sdk_version": "13.2.0",
		"level":              "info",
		"sdk_platform":       "android",
		"sdk_version":        "8.0.0",
	}, got)
}

func TestAppendOtelAttributesResolvesCollisionsDeterministically(t *testing.T) {
	want := []otellog.KeyValue{
		otellog.String("channel", "sdkLogs"),
		otellog.String("context.channel", "context channel"),
		otellog.String("sdk_version", "8.0.0"),
		otellog.String("fields.channel", "message channel"),
		otellog.String("fields.fields.sdk_version", "literal repeated namespace"),
		otellog.String("fields.sdk_version", "literal namespace"),
		otellog.String("fields.fields.fields.sdk_version", "13.2.0"),
	}

	for range 25 {
		attributes := make([]otellog.KeyValue, 0, len(want))
		keys := make(map[string]struct{}, len(want))
		attributes = appendOtelAttribute(attributes, keys, "channel", "metadata", "sdkLogs")
		attributes = appendOtelAttributes(attributes, keys, "context", map[string]any{
			"sdk_version": "8.0.0",
			"channel":     "context channel",
		})
		attributes = appendOtelAttributes(attributes, keys, "fields", map[string]any{
			"sdk_version":               "13.2.0",
			"fields.sdk_version":        "literal namespace",
			"fields.fields.sdk_version": "literal repeated namespace",
			"channel":                   "message channel",
		})

		assert.Equal(t, want, attributes)
	}
}
