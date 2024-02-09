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

func (i *noopInput) Data() <-chan *Message {
	return i.ch
}

func (i *noopInput) Run(context.Context) error {
	<-i.ch

	return nil
}

func (i *noopInput) Stop() {
	i.once.Do(func() {
		close(i.ch)
	})
}
