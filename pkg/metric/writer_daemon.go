package metric

import (
	"github.com/justtrackio/gosoline/pkg/clock"
)

type daemonWriter struct {
	clock   clock.Clock
	channel *metricChannel
}

func NewDaemonWriter(defaults ...*Datum) *daemonWriter {
	testClock := clock.NewRealClock()
	channel := ProviderMetricChannel()

	addMetricDefaults(defaults...)

	return NewDaemonWriterWithInterfaces(testClock, channel)
}

func NewDaemonWriterWithInterfaces(clock clock.Clock, channel *metricChannel) *daemonWriter {
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
