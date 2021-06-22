package metric

import (
	"github.com/jonboulle/clockwork"
)

type daemonWriter struct {
	clock   clockwork.Clock
	channel *metricChannel
}

func NewDaemonWriter(defaults ...*Datum) *daemonWriter {
	clock := clockwork.NewRealClock()
	channel := ProviderMetricChannel()

	addMetricDefaults(defaults...)

	return NewDaemonWriterWithInterfaces(clock, channel)
}

func NewDaemonWriterWithInterfaces(clock clockwork.Clock, channel *metricChannel) *daemonWriter {
	return &daemonWriter{
		clock:   clock,
		channel: channel,
	}
}

func (w daemonWriter) GetPriority() int {
	return PriorityLow
}

func (w daemonWriter) Write(batch Data) {
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

func (w daemonWriter) WriteOne(data *Datum) {
	w.Write(Data{data})
}
