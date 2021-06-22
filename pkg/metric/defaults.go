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
