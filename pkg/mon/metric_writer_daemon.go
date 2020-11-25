package mon

import (
	"github.com/jonboulle/clockwork"
)

type daemonWriter struct {
	clock   clockwork.Clock
	channel *metricChannel
}

func NewMetricDaemonWriter(defaults ...*MetricDatum) *daemonWriter {
	clock := clockwork.NewRealClock()
	channel := ProviderMetricChannel()

	addMetricDefaults(defaults...)

	return NewMetricDaemonWriterWithInterfaces(clock, channel)
}

func NewMetricDaemonWriterWithInterfaces(clock clockwork.Clock, channel *metricChannel) *daemonWriter {
	return &daemonWriter{
		clock:   clock,
		channel: channel,
	}
}

func (w daemonWriter) GetPriority() int {
	return PriorityLow
}

func (w daemonWriter) Write(batch MetricData) {
	if !w.channel.enabled || len(batch) == 0 {
		return
	}

	for i := 0; i < len(batch); i++ {
		if batch[i].Timestamp.IsZero() {
			batch[i].Timestamp = w.clock.Now()
		}
	}

	w.channel.write(batch)
}

func (w daemonWriter) WriteOne(data *MetricDatum) {
	w.Write(MetricData{data})
}
