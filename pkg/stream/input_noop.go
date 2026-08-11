package stream

import (
	"context"
	"sync"
)

type noopInput struct {
	ch   chan *Message
	once sync.Once
}

func NewNoopInput() Input {
	return &noopInput{
		ch: make(chan *Message),
	}
}

func (i *noopInput) Run(ctx context.Context, _ InputProcess) error {
	select {
	case <-ctx.Done():
	case <-i.ch:
	}

	return nil
}

func (i *noopInput) Stop(ctx context.Context) {
	i.once.Do(func() {
		close(i.ch)
	})
}

func (i *noopInput) IsHealthy() bool {
	return true
}
