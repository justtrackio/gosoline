package log

import (
	"testing"

	"github.com/stretchr/testify/assert"
	otellog "go.opentelemetry.io/otel/log"
)

func TestAppendOtelAttributesPreservesContextAndConflictingFields(t *testing.T) {
	attributes := make([]otellog.KeyValue, 0, 3)
	keys := make(map[string]struct{}, 3)

	attributes = appendOtelAttribute(attributes, keys, "channel", "metadata", "sdkLogs")
	attributes = appendOtelAttributes(attributes, keys, "context", map[string]any{
		"sdk_version": "8.0.0",
	})
	attributes = appendOtelAttributes(attributes, keys, "fields", map[string]any{
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
		"sdk_version":        "8.0.0",
	}, got)
}
