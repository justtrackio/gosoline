package metrics_per_runner

import (
	"context"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

var handlers = map[string]Handler{}

type Handler interface {
	IsEnabled(config cfg.Config) bool
	Init(ctx context.Context, config cfg.Config, logger log.Logger, cwNamespace string) (*HandlerSettings, error)
	GetMetricSum(ctx context.Context) (float64, error)
}

type HandlerSettings struct {
	MaxIncreasePercent float64       `cfg:"max_increase_percent" default:"200"`
	MaxIncreasePeriod  time.Duration `cfg:"max_increase_period" default:"5m"`
	Period             time.Duration `cfg:"period" default:"1m"`
	TargetValue        float64       `cfg:"target_value" default:"0"`
}

func RegisterHandler(name string, handler Handler) {
	if _, ok := handlers[name]; ok {
		panic("handler with name " + name + " already exists")
	}

	handlers[name] = handler
}
