package metric

import "sync"

var metricDefaultsLock = sync.Mutex{}
var metricDefaults = map[string]*Datum{}

func addMetricDefaults(data ...*Datum) {
	metricDefaultsLock.Lock()
	defer metricDefaultsLock.Unlock()

	for _, datum := range data {
		id := datum.Id()
		metricDefaults[id] = datum
	}
}

func amendFromDefault(datum *Datum) {
	defId := datum.Id()
	def, ok := metricDefaults[defId]

	if !ok {
		return
	}

	if datum.Priority == 0 {
		datum.Priority = def.Priority
	}

	if datum.Unit == "" {
		datum.Unit = def.Unit
	}
}
