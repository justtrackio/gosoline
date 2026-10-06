package log

import (
	"testing"

	"github.com/stretchr/testify/assert"
	otellog "go.opentelemetry.io/otel/log"
)

func TestAppendOtelAttributesAlwaysNamespacesSources(t *testing.T) {
	attributes := make([]otellog.KeyValue, 0, 3)
	keys := make(map[string]struct{}, 3)
	warnings := make([]string, 0)
	warnDuplicate := func(key string) {
		warnings = append(warnings, key)
	}

	attributes = appendOtelAttribute(attributes, keys, "metadata", "channel", "sdkLogs", warnDuplicate)
	attributes = appendOtelAttributes(attributes, keys, "context", map[string]any{
		"sdk_platform": "android",
		"sdk_version":  "8.0.0",
	}, warnDuplicate)
	attributes = appendOtelAttributes(attributes, keys, "fields", map[string]any{
		"level":       "info",
		"sdk_version": "13.2.0",
	}, warnDuplicate)

	got := make(map[string]string, len(attributes))
	gotKeys := make([]string, 0, len(attributes))
	for _, attribute := range attributes {
		gotKeys = append(gotKeys, attribute.Key)
		if assert.NotContains(t, got, attribute.Key) {
			got[attribute.Key] = attribute.Value.AsString()
		}
	}

	assert.Len(t, got, len(attributes))
	assert.Equal(t, map[string]string{
		"context.sdk_platform": "android",
		"context.sdk_version":  "8.0.0",
		"fields.level":         "info",
		"fields.sdk_version":   "13.2.0",
		"metadata.channel":     "sdkLogs",
	}, got)
	assert.Equal(t, []string{
		"metadata.channel",
		"context.sdk_platform",
		"context.sdk_version",
		"fields.level",
		"fields.sdk_version",
	}, gotKeys)
	assert.Empty(t, warnings)
}

func TestAppendOtelAttributeWarnsAndDropsDuplicateNamespacedKey(t *testing.T) {
	attributes := make([]otellog.KeyValue, 0, 2)
	keys := make(map[string]struct{}, 2)
	warnings := make([]string, 0)
	warnDuplicate := func(key string) {
		warnings = append(warnings, key)
	}

	attributes = appendOtelAttribute(attributes, keys, "fields", "sdk_version", "13.2.0", warnDuplicate)
	attributes = appendOtelAttribute(attributes, keys, "fields", "sdk_version", "14.0.0", warnDuplicate)

	assert.Len(t, attributes, 1)
	assert.Equal(t, "fields.sdk_version", attributes[0].Key)
	assert.Equal(t, "13.2.0", attributes[0].Value.AsString())
	assert.Equal(t, []string{"fields.sdk_version"}, warnings)
}
