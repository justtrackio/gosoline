package mon_test

import (
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
)

// ensure the metric daemon implements the full module interface
var _ kernel.FullModule = &mon.MetricDaemon{}
