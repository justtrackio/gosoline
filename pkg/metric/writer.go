package metric

import (
	"github.com/justtrackio/gosoline/pkg/clock"
)

type writer struct {
	clock   clock.Clock
	channel *metricChannel
}

func NewWriter(defaults ...*Datum) Writer {
	channel := ProviderMetricChannel()

	addMetricDefaults(defaults...)

	return NewWriterWithInterfaces(clock.Provider, channel)
}

func NewWriterWithInterfaces(clock clock.Clock, channel *metricChannel) Writer {
	return &writer{
		clock:   clock,
		channel: channel,
	}
}

func (w writer) GetPriority() int {
	return PriorityLow
}

func (w writer) Write(batch Data) {
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

func (w writer) WriteOne(data *Datum) {
	w.Write(Data{data})
}
