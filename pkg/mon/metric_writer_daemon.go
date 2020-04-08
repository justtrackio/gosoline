package mon

import (
	"github.com/jonboulle/clockwork"
	"sync"
	"sync/atomic"
	"unsafe"
)

type MetricDaemon interface {
	AddDefaults(defaults ...*MetricDatum)
	IsEnabled() bool
	Write(batch MetricData)
}

type pendingMetricDaemon struct {
	lck      sync.Mutex
	done     bool
	defaults []*MetricDatum
	written  MetricData
}

type metricContainer struct {
	daemon MetricDaemon
}

// to avoid having to lock the daemon every time we want to
// read it (a lot of times), we store it in a container which
// we can atomically read and swap. this is safe because if
// someone is using the pendingMetricDaemon while we swap it
// you either get the old or new value. if you get the old value,
// either you or the routine swapping the daemon will win the
// race to the mutex. if you win, you write your data and it
// will afterwards be forwarded to the new daemon. if you lose
// the race, you will see that done was already set and use the
// new daemon directly.
// after we did the swap we will always just load the new pointer
// and write to the new daemon.
var metricDaemon = unsafe.Pointer(&metricContainer{
	daemon: new(pendingMetricDaemon),
})

func (p *pendingMetricDaemon) AddDefaults(defaults ...*MetricDatum) {
	p.lck.Lock()
	defer p.lck.Unlock()

	if p.done {
		getMetricDaemon().AddDefaults(defaults...)

		return
	}

	p.defaults = append(p.defaults, defaults...)
}

func (p *pendingMetricDaemon) IsEnabled() bool {
	return !p.done || getMetricDaemon().IsEnabled()
}

func (p *pendingMetricDaemon) Write(batch MetricData) {
	p.lck.Lock()
	defer p.lck.Unlock()

	if p.done {
		getMetricDaemon().Write(batch)

		return
	}

	p.written = append(p.written, batch...)
}

func (p *pendingMetricDaemon) flush() {
	p.lck.Lock()
	p.done = true
	p.lck.Unlock()

	getMetricDaemon().AddDefaults(p.defaults...)
	getMetricDaemon().Write(p.written)

	p.defaults = nil
	p.written = nil
}

func getMetricDaemon() MetricDaemon {
	return (*metricContainer)(atomic.LoadPointer(&metricDaemon)).daemon
}

func InitializeMetricDaemon(daemon MetricDaemon) {
	old := (*metricContainer)(atomic.SwapPointer(&metricDaemon, unsafe.Pointer(&metricContainer{
		daemon: daemon,
	})))

	if pending, ok := old.daemon.(*pendingMetricDaemon); ok {
		pending.flush()
	}
}

type daemonWriter struct {
	clock clockwork.Clock
}

func NewMetricDaemonWriter(defaults ...*MetricDatum) *daemonWriter {
	clock := clockwork.NewRealClock()

	getMetricDaemon().AddDefaults(defaults...)

	return NewMetricDaemonWriterWithInterfaces(clock)
}

func NewMetricDaemonWriterWithInterfaces(clock clockwork.Clock) *daemonWriter {
	return &daemonWriter{
		clock: clock,
	}
}

func (w daemonWriter) GetPriority() int {
	return PriorityLow
}

func (w daemonWriter) Write(batch MetricData) {
	if !getMetricDaemon().IsEnabled() || len(batch) == 0 {
		return
	}

	for i := 0; i < len(batch); i++ {
		if batch[i].Timestamp.IsZero() {
			batch[i].Timestamp = w.clock.Now()
		}
	}

	getMetricDaemon().Write(batch)
}

func (w daemonWriter) WriteOne(data *MetricDatum) {
	w.Write(MetricData{data})
}
