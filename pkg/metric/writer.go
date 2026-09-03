package metric

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/clock"
)

const (
	PriorityLow  = 1
	PriorityHigh = 2

	DimensionDefault = "{{default}}"
)

//go:generate go run github.com/vektra/mockery/v2 --name Writer
type (
	Writer interface {
		GetPriority() int
		Write(ctx context.Context, batch Data)
		WriteOne(ctx context.Context, data *Datum)
	}

	writer struct {
		clock     clock.Clock
		channel   *metricChannel
		namespace string
	}
)

// NewWriter creates a writer stamping the given canonical namespace onto every datum written through
// it that does not carry one already. Pass an empty namespace to write metrics under their leaf name
// alone. The defaults are registered under the same namespace, so a datum written later resolves to
// the default authored for it.
func NewWriter(namespace string, defaults ...*Datum) Writer {
	channel := providerMetricChannel(func(*metricChannel) {})

	for _, datum := range defaults {
		if datum.Namespace == "" {
			datum.Namespace = namespace
		}
	}

	addMetricDefaults(defaults...)

	return NewWriterWithInterfaces(clock.Provider, channel, namespace)
}

func NewWriterWithInterfaces(clock clock.Clock, channel *metricChannel, namespace string) Writer {
	return &writer{
		clock:     clock,
		channel:   channel,
		namespace: namespace,
	}
}

func (w writer) GetPriority() int {
	return PriorityLow
}

func (w writer) Write(_ context.Context, batch Data) {
	if !w.channel.enabled || len(batch) == 0 {
		return
	}

	for i := 0; i < len(batch); i++ {
		if batch[i].Timestamp.IsZero() {
			batch[i].Timestamp = w.clock.Now()
		}

		if batch[i].Namespace == "" {
			batch[i].Namespace = w.namespace
		}
	}

	w.channel.write(batch)
}

func (w writer) WriteOne(ctx context.Context, data *Datum) {
	w.Write(ctx, Data{data})
}
