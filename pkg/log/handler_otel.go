package log

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/otel"
	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

const otelLoggerName = "github.com/justtrackio/gosoline/pkg/log"

func init() {
	AddHandlerFactory("otel", handlerOtelFactory)
}

// HandlerOtelSettings configures the "otel" log handler, which exports logs to an OTEL collector
// via OTLP using the shared otel.* configuration. Logs carry trace/span context for correlation.
type HandlerOtelSettings struct {
	Level string `cfg:"level" default:"info"`
}

func handlerOtelFactory(config cfg.Config, name string) (Handler, error) {
	settings := &HandlerOtelSettings{}
	if err := UnmarshalHandlerSettingsFromConfig(config, name, settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal otel handler settings: %w", err)
	}

	priority, ok := LevelPriority(settings.Level)
	if !ok {
		return nil, fmt.Errorf("invalid log level %q", settings.Level)
	}

	otelSettings, err := otel.ReadSettings(config)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	res, err := otel.BuildResource(config, otelSettings.Resource)
	if err != nil {
		return nil, fmt.Errorf("could not build otel resource: %w", err)
	}

	exporter, err := otel.BuildLogExporter(ctx, otelSettings.Exporter)
	if err != nil {
		return nil, fmt.Errorf("could not build otel log exporter: %w", err)
	}

	provider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	)

	return NewHandlerOtel(config, priority, name, provider), nil
}

type handlerOtel struct {
	handlerBase
	provider *sdklog.LoggerProvider
	logger   otellog.Logger
}

// NewHandlerOtel creates a handler backed by the provided OpenTelemetry logger provider.
func NewHandlerOtel(config cfg.Config, levelPriority int, name string, provider *sdklog.LoggerProvider) Handler {
	return &handlerOtel{
		handlerBase: handlerBase{
			config:   config,
			level:    levelPriority,
			channels: make(map[string]*int),
			name:     name,
		},
		provider: provider,
		logger:   provider.Logger(otelLoggerName),
	}
}

// Log builds an OTEL LogRecord and emits it. The OTEL Logs SDK extracts the active span context
// from ctx and attaches trace_id/span_id to the exported record, enabling trace<->log correlation.
func (h *handlerOtel) Log(ctx context.Context, timestamp time.Time, level int, msg string, args []any, logErr error, data Data) error {
	body := msg
	if len(args) > 0 {
		body = fmt.Sprintf(msg, args...)
	}

	var record otellog.Record
	record.SetTimestamp(timestamp)
	record.SetSeverity(toOtelSeverity(level))
	record.SetSeverityText(LevelName(level))
	record.SetBody(otellog.StringValue(body))

	attributeCapacity := len(data.Fields) + len(data.ContextFields) + 2
	attributes := make([]otellog.KeyValue, 0, attributeCapacity)
	attributeKeys := make(map[string]struct{}, attributeCapacity)
	// Context keys stay flat; message keys always use fields., independently of collisions.
	// Framework keys and the fields. prefix are reserved so sources cannot overlap.
	attributes = appendOtelAttribute(attributes, attributeKeys, "channel", "", data.Channel, os.Stderr)
	attributes = appendOtelAttributes(attributes, attributeKeys, "", data.ContextFields, os.Stderr)
	attributes = appendOtelAttributes(attributes, attributeKeys, "fields", data.Fields, os.Stderr)

	if logErr != nil {
		attributes = appendOtelAttribute(attributes, attributeKeys, "error", "", logErr.Error(), os.Stderr)
	}

	record.AddAttributes(attributes...)

	h.logger.Emit(ctx, record)

	return nil
}

// appendOtelAttributes assigns names by source and sorts keys for stable output.
// An empty namespace denotes context fields, which cannot use reserved keys.
func appendOtelAttributes(attributes []otellog.KeyValue, keys map[string]struct{}, namespace string, values map[string]any, warnings io.Writer) []otellog.KeyValue {
	valueKeys := make([]string, 0, len(values))
	for key := range values {
		valueKeys = append(valueKeys, key)
	}
	sort.Strings(valueKeys)

	for _, key := range valueKeys {
		if namespace == "" && (key == "channel" || key == "error" || strings.HasPrefix(key, "fields.")) {
			// Write directly to avoid recursively invoking this log handler.
			_, _ = fmt.Fprintf(warnings, "Warning: dropping OTel context attribute %q: key is reserved\n", key) //nolint:errcheck // Diagnostics must not prevent exporting the original log record.

			continue
		}

		attributes = appendOtelAttribute(attributes, keys, key, namespace, values[key], warnings)
	}

	return attributes
}

func appendOtelAttribute(attributes []otellog.KeyValue, keys map[string]struct{}, key, namespace string, value any, warnings io.Writer) []otellog.KeyValue {
	if namespace != "" {
		key = namespace + "." + key
	}

	if _, exists := keys[key]; exists {
		// Duplicates indicate a bug: do not give the same field another name.
		_, _ = fmt.Fprintf(warnings, "Warning: duplicate OTel log attribute %q ignored\n", key) //nolint:errcheck // Diagnostics must not prevent exporting the original log record.

		return attributes
	}

	keys[key] = struct{}{}

	return append(attributes, toOtelKeyValue(key, value))
}

func (h *handlerOtel) Close(ctx context.Context) error {
	return h.provider.Shutdown(ctx)
}

func toOtelSeverity(level int) otellog.Severity {
	switch level {
	case PriorityTrace:
		return otellog.SeverityTrace
	case PriorityDebug:
		return otellog.SeverityDebug
	case PriorityInfo:
		return otellog.SeverityInfo
	case PriorityWarn:
		return otellog.SeverityWarn
	case PriorityError:
		return otellog.SeverityError
	default:
		return otellog.SeverityUndefined
	}
}

func toOtelKeyValue(key string, value any) otellog.KeyValue {
	switch v := value.(type) {
	case string:
		return otellog.String(key, v)
	case bool:
		return otellog.Bool(key, v)
	case int:
		return otellog.Int64(key, int64(v))
	case int64:
		return otellog.Int64(key, v)
	case float64:
		return otellog.Float64(key, v)
	case float32:
		return otellog.Float64(key, float64(v))
	default:
		return otellog.String(key, fmt.Sprintf("%v", v))
	}
}
