package tracing

import (
	"time"

	"github.com/justtrackio/gosoline/pkg/log"
)

type ContextMissingWarnSamplingConfig struct {
	Enabled  bool
	Interval time.Duration
}

type ContextMissingWarnStrategy struct {
	logger   log.Logger
	sampling ContextMissingWarnSamplingConfig
}

func NewContextMissingWarningLogStrategy(logger log.Logger) *ContextMissingWarnStrategy {
	logger = logger.WithChannel("tracing")
	logger = log.NewSamplingLogger(logger, time.Minute)

	strategy := &ContextMissingWarnStrategy{
		logger: logger,
		sampling: ContextMissingWarnSamplingConfig{
			Enabled:  true,
			Interval: time.Minute,
		},
	}

	return strategy
}

func (c ContextMissingWarnStrategy) ContextMissing(v any) {
	stacktrace := log.GetStackTrace(2)

	c.logger.WithFields(log.Fields{
		"stacktrace": stacktrace,
	}).Warn("can not trace the action: %s", v)
}
