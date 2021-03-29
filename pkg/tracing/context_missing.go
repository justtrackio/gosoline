package tracing

import (
	"github.com/applike/gosoline/pkg/mon"
	"time"
)

type ContextMissingWarnSamplingConfig struct {
	Enabled  bool
	Interval time.Duration
}

type ContextMissingWarnStrategy struct {
	logger   mon.Logger
	sampling ContextMissingWarnSamplingConfig
}

func NewContextMissingWarningLogStrategy(logger mon.Logger) *ContextMissingWarnStrategy {
	logger = logger.WithChannel("tracing")
	logger = mon.NewSamplingLogger(logger, time.Minute)

	strategy := &ContextMissingWarnStrategy{
		logger: logger,
		sampling: ContextMissingWarnSamplingConfig{
			Enabled:  true,
			Interval: time.Minute,
		},
	}

	return strategy
}

func (c ContextMissingWarnStrategy) ContextMissing(v interface{}) {
	stacktrace := mon.GetStackTrace(2)

	c.logger.WithFields(mon.Fields{
		"stacktrace": stacktrace,
	}).Warn("can not trace the action: %s", v)
}
