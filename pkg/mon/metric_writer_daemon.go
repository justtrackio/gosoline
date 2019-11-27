package mon

import (
	"github.com/jonboulle/clockwork"
)

type daemonWriter struct {
	clock  clockwork.Clock
	daemon *cwDaemon
}

func NewMetricDaemonWriter(defaults ...*MetricDatum) *daemonWriter {
	clock := clockwork.NewRealClock()
	daemon := ProvideCwDaemon()

	daemon.AddDefaults(defaults...)

	return NewMetricDaemonWriterWithInterfaces(clock, daemon)
}

func NewMetricDaemonWriterWithInterfaces(clock clockwork.Clock, daemon *cwDaemon) *daemonWriter {
	return &daemonWriter{
		clock:  clock,
		daemon: daemon,
	}
}

func (w daemonWriter) GetPriority() int {
	return PriorityLow
}

func (w daemonWriter) Write(batch MetricData) {
	if !w.daemon.settings.Enabled {
		return
	}

	for i := 0; i < len(batch); i++ {
		if batch[i].Timestamp.IsZero() {
			batch[i].Timestamp = w.clock.Now()
		}

		w.daemon.channel <- batch[i]
	}
}

func (w daemonWriter) WriteOne(data *MetricDatum) {
	w.Write(MetricData{data})
}
