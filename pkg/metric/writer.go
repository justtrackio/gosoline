package metric

import (
	"github.com/justtrackio/gosoline/pkg/clock"
)

const (
	PriorityLow  = 1
	PriorityHigh = 2

	KindTotal   Kind = "total"
	KindDefault Kind = ""

	DimensionDefault = "{{default}}"
)

//go:generate mockery --name Writer
type (
	Writer interface {
		GetPriority() int
		Write(batch Data)
		WriteOne(data *Datum)
	}

	writer struct {
		clock   clock.Clock
		channel *metricChannel
	}

	Kind string
)

func NewWriter(defaults ...*Datum) Writer {
	channel := providerMetricChannel(func(*metricChannel) {})

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
