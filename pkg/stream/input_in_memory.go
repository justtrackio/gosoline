package stream

import (
	"context"
	"sync"
)

var inMemoryInputs = make(map[string]*InMemoryInput)

func ResetInMemoryInputs() {
	for _, inp := range inMemoryInputs {
		inp.Reset()
	}
}

type InMemorySettings struct {
	Size int `cfg:"size" default:"1"`
}

type InMemoryInput struct {
	once     sync.Once
	channel  chan *Message
	stopped  chan struct{}
	settings *InMemorySettings
}

func ProvideInMemoryInput(name string, settings *InMemorySettings) *InMemoryInput {
	if input, ok := inMemoryInputs[name]; ok {
		return input
	}

	inMemoryInputs[name] = NewInMemoryInput(settings)

	return inMemoryInputs[name]
}

func NewInMemoryInput(settings *InMemorySettings) *InMemoryInput {
	return &InMemoryInput{
		channel:  make(chan *Message, settings.Size),
		stopped:  make(chan struct{}),
		settings: settings,
	}
}

func (i *InMemoryInput) Reset() {
	i.once = sync.Once{}
	i.channel = make(chan *Message, i.settings.Size)
	i.stopped = make(chan struct{})
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
