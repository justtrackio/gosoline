package mon

import "sync"

var metricDefaultsLock = sync.Mutex{}
var metricDefaults = make(map[string]*MetricDatum, 0)

func addMetricDefaults(data ...*MetricDatum) {
	metricDefaultsLock.Lock()
	defer metricDefaultsLock.Unlock()

	for _, datum := range data {
		id := datum.Id()
		metricDefaults[id] = datum
	}
}
