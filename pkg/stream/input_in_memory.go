package stream

import (
	"context"
	"sync"
)

var inMemoryInputs = make(map[string]*InMemoryInput)

type InMemorySettings struct {
	Size int `cfg:"size" default:"1"`
}

type InMemoryInput struct {
	once    sync.Once
	channel chan *Message
	stopped chan struct{}
}

func ProvideInMemoryInput(name string, settings *InMemorySettings) *InMemoryInput {
	if input, ok := inMemoryInputs[name]; ok {
		return input
	}

	inMemoryInputs[name] = &InMemoryInput{
		channel: make(chan *Message, settings.Size),
		stopped: make(chan struct{}),
	}

	return inMemoryInputs[name]
}

func (i *InMemoryInput) Publish(messages ...*Message) {
	for _, msg := range messages {
		i.channel <- msg
	}
}

func (i *InMemoryInput) Run(ctx context.Context) error {
	select {
	case <-ctx.Done():
	case <-i.stopped:
	}

	close(i.channel)
	return nil
}

func (i *InMemoryInput) Stop() {
	i.once.Do(func() {
		close(i.stopped)
	})
}

func (i *InMemoryInput) Data() chan *Message {
	return i.channel
}
