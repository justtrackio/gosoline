package metric

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"time"
)

func init() {
	log.AddHandlerFactory("metric", LoggerHandlerFactory)
}

func LoggerHandlerFactory(_ cfg.Config, _ string) (log.Handler, error) {
	return NewLoggerHandler(), nil
}

func NewLoggerHandler() *LoggerHandler {
	defaults := getDefaultMetrics()
	writer := NewDaemonWriter(defaults...)

	return &LoggerHandler{
		writer: writer,
	}
}

type LoggerHandler struct {
	writer Writer
}

func (h LoggerHandler) Channels() []string {
	return []string{}
}

func (h LoggerHandler) Level() int {
	return log.PriorityWarn
}

func (h LoggerHandler) Log(_ time.Time, level int, _ string, _ []interface{}, _ error, _ log.Data) error {
	if level != log.PriorityWarn && level != log.PriorityError {
		return nil
	}

	h.writer.WriteOne(&Datum{
		Priority:   PriorityHigh,
		MetricName: log.LevelName(level),
		Unit:       UnitCount,
		Value:      1.0,
	})

	return nil
}

func getDefaultMetrics() Data {
	return Data{
		{
			Priority:   PriorityHigh,
			MetricName: log.LevelError,
			Unit:       UnitCount,
			Value:      0.0,
		},
		{
			Priority:   PriorityHigh,
			MetricName: log.LevelWarn,
			Unit:       UnitCount,
			Value:      0.0,
		},
	}
}
