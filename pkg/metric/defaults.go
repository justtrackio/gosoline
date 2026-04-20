package metric

import "sync"

var (
	metricDefaultsLock sync.RWMutex
	metricDefaults     = map[string]*Datum{}
)

func addMetricDefaults(data ...*Datum) {
	metricDefaultsLock.Lock()
	defer metricDefaultsLock.Unlock()

	for _, datum := range data {
		id := datum.Id()
		metricDefaults[id] = datum
	}
}

func amendFromDefault(datum *Datum) {
	metricDefaultsLock.RLock()
	defId := datum.Id()
	def, ok := metricDefaults[defId]
	metricDefaultsLock.RUnlock()

	if !ok {
		return
	}

	if datum.Priority == 0 {
		datum.Priority = def.Priority
	}

	if datum.Unit == "" {
		datum.Unit = def.Unit
	}

	if datum.Kind.kind == "" {
		datum.Kind = def.Kind
	}
}
