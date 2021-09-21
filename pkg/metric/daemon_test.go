package metric_test

import (
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/metric"
)

// ensure the metric daemon implements the full module interface
var _ kernel.FullModule = &metric.Daemon{}
