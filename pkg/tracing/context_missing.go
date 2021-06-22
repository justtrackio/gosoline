package tracing

import (
	"github.com/applike/gosoline/pkg/log"
	"time"
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

func (c ContextMissingWarnStrategy) ContextMissing(v interface{}) {
	stacktrace := log.GetStackTrace(2)

	c.logger.WithFields(log.Fields{
		"stacktrace": stacktrace,
	}).Warn("can not trace the action: %s", v)
}
