package metric

import (
	"context"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

func init() {
	log.AddHandlerFactory("metric", LoggerHandlerFactory)
}

const (
	metricNameLogRecords = "records"

	// DimensionLogLevel names the level a log record was written at.
	DimensionLogLevel = "log.level"
)

func LoggerHandlerFactory(_ cfg.Config, _ string) (log.Handler, error) {
	return NewLoggerHandler(), nil
}

func NewLoggerHandler() *LoggerHandler {
	defaults := getDefaultMetrics()
	metricWriter := NewWriter(NamespaceMetric, defaults...)

	return &LoggerHandler{
		writer: metricWriter,
	}
}

type LoggerHandler struct {
	writer Writer
}

func (h LoggerHandler) ChannelLevel(string) (level *int, err error) {
	return nil, nil
}

func (h LoggerHandler) Level() int {
	return log.PriorityWarn
}

func (h LoggerHandler) Log(ctx context.Context, _ time.Time, level int, _ string, _ []any, _ error, _ log.Data) error {
	if level != log.PriorityWarn && level != log.PriorityError {
		return nil
	}

	h.writer.WriteOne(ctx, logRecordDatum(log.LevelName(level), 1.0))

	return nil
}

func getDefaultMetrics() Data {
	return Data{
		defaultLogRecordDatum(log.LevelError, 0.0),
		defaultLogRecordDatum(log.LevelWarn, 0.0),
	}
}

func defaultLogRecordDatum(level string, value float64) *Datum {
	datum := logRecordDatum(level, value)
	datum.Unit = UnitCount
	datum.Kind = KindCounter.Build()

	return datum
}

// logRecordDatum counts a log record. The level is a dimension rather than part of the name, so one
// metric answers "how much is this service logging" across every level.
func logRecordDatum(level string, value float64) *Datum {
	return &Datum{
		Priority:   PriorityHigh,
		MetricName: metricNameLogRecords,
		Dimensions: Dimensions{
			DimensionLogLevel: level,
		},
		Value: value,
	}
}
