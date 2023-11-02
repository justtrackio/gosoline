package metric_test

import (
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/metric"
)

// ensure the metric daemon implements the typed and staged module interfaces
var _ interface {
	kernel.Module
	kernel.TypedModule
	kernel.StagedModule
} = &metric.Daemon{}
