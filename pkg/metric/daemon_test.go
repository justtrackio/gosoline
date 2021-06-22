package metric_test

import (
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/metric"
)

// ensure the metric daemon implements the full module interface
var _ kernel.FullModule = &metric.Daemon{}
