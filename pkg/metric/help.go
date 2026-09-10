package metric

import (
	"fmt"
	"sync"
)

var (
	metricHelpLock sync.RWMutex
	metricHelps    = map[string]string{}
)

// RegisterHelp declares the help text the metric authored under namespace and metricName is exported
// with. The emitting package registers it once, next to the name it owns, so the description does not
// have to be repeated on every datum written for that metric. Pass a help on the datum's Kind, via
// WithHelp, to describe a single series more specifically than its metric.
//
// Registering the same help twice is a no-op, which is what lets several packages emit one shared
// metric. Registering a different help for a name already registered panics: a backend keeps one
// description per metric name, so the second description would be rejected when the metric is
// registered and that metric would silently stop being exported.
func RegisterHelp(namespace, metricName, help string) {
	name := canonicalName(namespace, metricName)

	if help == "" {
		panic(fmt.Errorf("can not register an empty help for metric %s", name))
	}

	metricHelpLock.Lock()
	defer metricHelpLock.Unlock()

	if registered, ok := metricHelps[name]; ok && registered != help {
		panic(fmt.Errorf("can not register help %q for metric %s, it is already registered as %q", help, name, registered))
	}

	metricHelps[name] = help
}

// resolveHelp returns the help text a datum is exported with: the one its own Kind carries, else the
// one registered for its canonical name, else a description of its unit. The unit fallback is what a
// metric authored outside gosoline is exported with, since no package registered a help for it.
func resolveHelp(datum *Datum) string {
	if datum.Kind.help != "" {
		return datum.Kind.help
	}

	metricHelpLock.RLock()
	help, ok := metricHelps[canonicalName(datum.Namespace, datum.MetricName)]
	metricHelpLock.RUnlock()

	if ok {
		return help
	}

	return fmt.Sprintf("unit: %s", datum.Unit)
}
