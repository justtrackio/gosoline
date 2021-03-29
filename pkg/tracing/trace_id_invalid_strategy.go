package tracing

import (
	"github.com/applike/gosoline/pkg/mon"
	"time"
)

type TraceIdErrorStrategy interface {
	TraceIdInvalid(err error) error
}

type TraceIdErrorReturnStrategy struct{}

func (t TraceIdErrorReturnStrategy) TraceIdInvalid(err error) error {
	return err
}

type TraceIdErrorWarningStrategy struct {
	logger             mon.Logger
	stacktraceProvider mon.StackTraceProvider
}

func NewTraceIdErrorWarningStrategy(logger mon.Logger) *TraceIdErrorWarningStrategy {
	logger = logger.WithChannel("tracing")
	logger = mon.NewSamplingLogger(logger, time.Minute)

	return NewTraceIdErrorWarningStrategyWithInterfaces(logger, mon.GetStackTrace)
}

func NewTraceIdErrorWarningStrategyWithInterfaces(logger mon.Logger, stacktraceProvider mon.StackTraceProvider) *TraceIdErrorWarningStrategy {
	return &TraceIdErrorWarningStrategy{
		logger:             logger,
		stacktraceProvider: stacktraceProvider,
	}
}

func (t TraceIdErrorWarningStrategy) TraceIdInvalid(err error) error {
	stacktrace := t.stacktraceProvider(2)

	t.logger.WithFields(mon.Fields{
		"stacktrace": stacktrace,
	}).Warn("trace id is invalid: %s", err.Error())

	return nil
}
