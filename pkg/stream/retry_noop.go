package stream

import (
	"context"
	"sync"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

func init() {
	retryHandlers["noop"] = NewRetryHandlerNoop
}

type RetryHandlerNoop struct {
	ch   chan *Message
	once sync.Once
}

func NewRetryHandlerNoop(ctx context.Context, config cfg.Config, logger log.Logger, name string) (RetryHandler, error) {
	return NewRetryHandlerNoopWithInterfaces(), nil
}

func NewRetryHandlerNoopWithInterfaces() *RetryHandlerNoop {
	return &RetryHandlerNoop{
		ch: make(chan *Message),
	}
}

func (r *RetryHandlerNoop) Put(ctx context.Context, msg *Message) error {
	return nil
}

func (r *RetryHandlerNoop) Data() <-chan *Message {
	return r.ch
}

func (r *RetryHandlerNoop) Run(ctx context.Context) error {
	<-r.ch
	return nil
}

func (r *RetryHandlerNoop) Stop() {
	r.once.Do(func() {
		close(r.ch)
	})
}
