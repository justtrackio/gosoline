package stream

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

func init() {
	retryHandlers["noop"] = NewRetryHandlerNoop
}

type RetryHandlerNoop struct {
}

func NewRetryHandlerNoop(context.Context, cfg.Config, log.Logger, string) (Input, RetryHandler, error) {
	return NewNoopInput(), NewRetryHandlerNoopWithInterfaces(), nil
}

func NewRetryHandlerNoopWithInterfaces() RetryHandlerNoop {
	return RetryHandlerNoop{}
}

func (r RetryHandlerNoop) Put(context.Context, *Message) error {
	return nil
}
