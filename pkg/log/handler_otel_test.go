package log

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

func TestHandlerOtelNamespacesAttributesBySource(t *testing.T) {
	processor := &captureLogProcessor{}
	provider := sdklog.NewLoggerProvider(sdklog.WithProcessor(processor))
	handler := NewHandlerOtel(nil, PriorityInfo, "test", provider)

	err := handler.Log(context.Background(), time.Now(), PriorityInfo, "message", nil, errors.New("failure"), Data{
		Channel: "app",
		ContextFields: map[string]any{
			"channel":           "context channel",
			"error":             "context error",
			"fields.custom_key": "nested-context-value",
			"custom_key":        "context-value",
		},
		Fields: map[string]any{
			"channel":    "message channel",
			"error":      "message error",
			"custom_key": "field-value",
		},
	})
	require.NoError(t, err)
	require.NoError(t, provider.Shutdown(context.Background()))

	attributes := make(map[string]string, processor.record.AttributesLen())
	processor.record.WalkAttributes(func(attribute otellog.KeyValue) bool {
		attributes[attribute.Key] = attribute.Value.AsString()

		return true
	})

	assert.Equal(t, map[string]string{
		"channel":                   "app",
		"context.channel":           "context channel",
		"context.error":             "context error",
		"context.fields.custom_key": "nested-context-value",
		"context.custom_key":        "context-value",
		"fields.channel":            "message channel",
		"fields.error":              "message error",
		"fields.custom_key":         "field-value",
		"error":                     "failure",
	}, attributes)
}

type captureLogProcessor struct {
	record sdklog.Record
}

func (p *captureLogProcessor) Enabled(context.Context, sdklog.EnabledParameters) bool {
	return true
}

func (p *captureLogProcessor) OnEmit(_ context.Context, record *sdklog.Record) error {
	p.record = record.Clone()

	return nil
}

func (p *captureLogProcessor) Shutdown(context.Context) error {
	return nil
}

func (p *captureLogProcessor) ForceFlush(context.Context) error {
	return nil
}
