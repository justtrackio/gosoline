package tracing

import (
	"github.com/applike/gosoline/pkg/log"
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
	logger             log.Logger
	stacktraceProvider log.StackTraceProvider
}

func NewTraceIdErrorWarningStrategy(logger log.Logger) *TraceIdErrorWarningStrategy {
	logger = logger.WithChannel("tracing")
	logger = log.NewSamplingLogger(logger, time.Minute)

	return NewTraceIdErrorWarningStrategyWithInterfaces(logger, log.GetStackTrace)
}

func NewTraceIdErrorWarningStrategyWithInterfaces(logger log.Logger, stacktraceProvider log.StackTraceProvider) *TraceIdErrorWarningStrategy {
	return &TraceIdErrorWarningStrategy{
		logger:             logger,
		stacktraceProvider: stacktraceProvider,
	}
}

func (t TraceIdErrorWarningStrategy) TraceIdInvalid(err error) error {
	stacktrace := t.stacktraceProvider(2)

	t.logger.WithFields(log.Fields{
		"stacktrace": stacktrace,
	}).Warn("trace id is invalid: %s", err.Error())

	return nil
}
