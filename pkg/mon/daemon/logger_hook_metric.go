package daemon

import "github.com/applike/gosoline/pkg/mon"

type metricHook struct {
	writer      mon.MetricWriter
	application string
}

func NewMetricHook() *metricHook {
	defaults := getDefaultMetrics()
	writer := mon.NewMetricDaemonWriter(defaults...)

	return &metricHook{
		writer: writer,
	}
}

func (h metricHook) Fire(level string, _ string, _ error, _ *mon.Metadata) error {
	if level != mon.Warn && level != mon.Error {
		return nil
	}

	h.writer.WriteOne(&mon.MetricDatum{
		Priority:   mon.PriorityHigh,
		MetricName: level,
		Unit:       mon.UnitCount,
		Value:      1.0,
	})

	return nil
}

func getDefaultMetrics() mon.MetricData {
	return mon.MetricData{
		{
			Priority:   mon.PriorityHigh,
			MetricName: mon.Warn,
			Unit:       mon.UnitCount,
			Value:      0.0,
		},
		{
			Priority:   mon.PriorityHigh,
			MetricName: mon.Error,
			Unit:       mon.UnitCount,
			Value:      0.0,
		},
	}
}
