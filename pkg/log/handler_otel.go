package log

import (
	"context"
	"fmt"
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

func handlerOtelFactory(ctx context.Context, config cfg.Config, name string) (Handler, error) {
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

	setShutdownFn(ctx, provider.Shutdown)

	return NewHandlerOtel(config, priority, name, provider.Logger(otelLoggerName)), nil
}

type handlerOtel struct {
	handlerBase
	logger otellog.Logger
}

func NewHandlerOtel(config cfg.Config, levelPriority int, name string, logger otellog.Logger) Handler {
	return &handlerOtel{
		handlerBase: handlerBase{
			config:   config,
			level:    levelPriority,
			channels: make(map[string]*int),
			name:     name,
		},
		logger: logger,
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

	attributes := make([]otellog.KeyValue, 0, len(data.Fields)+len(data.ContextFields)+1)
	attributes = append(attributes, otellog.String("channel", data.Channel))

	for key, value := range data.ContextFields {
		attributes = append(attributes, toOtelKeyValue(key, value))
	}

	for key, value := range data.Fields {
		attributes = append(attributes, toOtelKeyValue(key, value))
	}

	if logErr != nil {
		attributes = append(attributes, otellog.String("error", logErr.Error()))
	}

	record.AddAttributes(attributes...)

	h.logger.Emit(ctx, record)

	return nil
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
