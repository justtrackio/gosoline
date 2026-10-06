package log

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	otellog "go.opentelemetry.io/otel/log"
)

func TestAppendOtelAttributesNamesBySource(t *testing.T) {
	tests := []struct {
		name    string
		context map[string]any
		fields  map[string]any
		want    map[string]string
	}{
		{
			name:    "context only",
			context: map[string]any{"sdk_version": "8.0.0"},
			want:    map[string]string{"channel": "sdkLogs", "sdk_version": "8.0.0"},
		},
		{
			name:   "message only",
			fields: map[string]any{"sdk_version": "13.2.0"},
			want:   map[string]string{"channel": "sdkLogs", "fields.sdk_version": "13.2.0"},
		},
		{
			name: "both sources",
			context: map[string]any{
				"sdk_platform": "android",
				"sdk_version":  "8.0.0",
			},
			fields: map[string]any{
				"level":       "info",
				"sdk_version": "13.2.0",
			},
			want: map[string]string{
				"channel":            "sdkLogs",
				"sdk_platform":       "android",
				"sdk_version":        "8.0.0",
				"fields.level":       "info",
				"fields.sdk_version": "13.2.0",
			},
		},
		{
			name: "literal message prefixes",
			fields: map[string]any{
				"sdk_version":               "13.2.0",
				"fields.sdk_version":        "literal namespace",
				"fields.fields.sdk_version": "literal repeated namespace",
			},
			want: map[string]string{
				"channel":                          "sdkLogs",
				"fields.sdk_version":               "13.2.0",
				"fields.fields.sdk_version":        "literal namespace",
				"fields.fields.fields.sdk_version": "literal repeated namespace",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var warnings bytes.Buffer
			keys := make(map[string]struct{})
			attributes := appendOtelAttribute(nil, keys, "channel", "", "sdkLogs", &warnings)
			attributes = appendOtelAttributes(attributes, keys, "", tt.context, &warnings)
			attributes = appendOtelAttributes(attributes, keys, "fields", tt.fields, &warnings)

			assert.Equal(t, tt.want, otelStringAttributes(t, attributes))
			assert.Empty(t, warnings.String())
		})
	}
}

func TestAppendOtelAttributesNamesIndependentOfSourceOrder(t *testing.T) {
	context := map[string]any{"sdk_version": "8.0.0", "sdk_platform": "android"}
	fields := map[string]any{"sdk_version": "13.2.0", "level": "info"}
	want := map[string]string{
		"sdk_platform":       "android",
		"sdk_version":        "8.0.0",
		"fields.level":       "info",
		"fields.sdk_version": "13.2.0",
	}

	for _, messageFirst := range []bool{false, true} {
		var warnings bytes.Buffer
		keys := make(map[string]struct{})
		var attributes []otellog.KeyValue
		if messageFirst {
			attributes = appendOtelAttributes(attributes, keys, "fields", fields, &warnings)
			attributes = appendOtelAttributes(attributes, keys, "", context, &warnings)
		} else {
			attributes = appendOtelAttributes(attributes, keys, "", context, &warnings)
			attributes = appendOtelAttributes(attributes, keys, "fields", fields, &warnings)
		}

		assert.Equal(t, want, otelStringAttributes(t, attributes))
		assert.Empty(t, warnings.String())
	}
}

func TestAppendOtelAttributesSortsMapKeys(t *testing.T) {
	want := []otellog.KeyValue{
		otellog.String("fields.level", "info"),
		otellog.String("fields.sdk_version", "13.2.0"),
	}

	for range 25 {
		attributes := appendOtelAttributes(nil, make(map[string]struct{}), "fields", map[string]any{
			"sdk_version": "13.2.0",
			"level":       "info",
		}, io.Discard)

		assert.Equal(t, want, attributes)
	}
}

func TestAppendOtelAttributesRejectsReservedContextKeys(t *testing.T) {
	for _, key := range []string{"fields.sdk_version", "fields.other", "channel", "error"} {
		t.Run(key, func(t *testing.T) {
			// Reserved keys are rejected even when no competing attribute is present.
			var warnings bytes.Buffer
			keys := make(map[string]struct{})
			attributes := appendOtelAttributes(nil, keys, "", map[string]any{
				key:           "invalid context value",
				"sdk_version": "8.0.0",
			}, &warnings)

			assert.Equal(t, map[string]string{"sdk_version": "8.0.0"}, otelStringAttributes(t, attributes))
			assert.NotContains(t, keys, key)
			assert.Equal(t, "Warning: dropping OTel context attribute \""+key+"\": key is reserved\n", warnings.String())

			attributes = appendOtelAttributes(attributes, keys, "fields", map[string]any{"sdk_version": "13.2.0"}, io.Discard)
			attributes = appendOtelAttribute(attributes, keys, "channel", "", "sdkLogs", io.Discard)
			attributes = appendOtelAttribute(attributes, keys, "error", "", "actual error", io.Discard)
			assert.Equal(t, map[string]string{
				"sdk_version":        "8.0.0",
				"fields.sdk_version": "13.2.0",
				"channel":            "sdkLogs",
				"error":              "actual error",
			}, otelStringAttributes(t, attributes))
		})
	}
}

func TestAppendOtelAttributeWarnsOnUnexpectedDuplicate(t *testing.T) {
	for _, namespace := range []string{"", "fields"} {
		t.Run(namespace, func(t *testing.T) {
			var warnings bytes.Buffer
			keys := make(map[string]struct{})
			attributes := appendOtelAttribute(nil, keys, "sdk_version", namespace, "original", &warnings)
			attributes = appendOtelAttribute(attributes, keys, "sdk_version", namespace, "duplicate", &warnings)
			key := "sdk_version"
			if namespace != "" {
				key = namespace + "." + key
			}

			assert.Equal(t, map[string]string{key: "original"}, otelStringAttributes(t, attributes))
			assert.Equal(t, "Warning: duplicate OTel log attribute \""+key+"\" ignored\n", warnings.String())
		})
	}
}

func otelStringAttributes(t *testing.T, attributes []otellog.KeyValue) map[string]string {
	t.Helper()

	got := make(map[string]string, len(attributes))
	for _, attribute := range attributes {
		require.NotContains(t, got, attribute.Key, "OTel attributes must have unique keys")
		got[attribute.Key] = attribute.Value.AsString()
	}

	return got
}
