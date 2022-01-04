package stream

import (
	"context"
	"sync"
)

var inMemoryInputsLock sync.Mutex
var inMemoryInputs = make(map[string]*InMemoryInput)

func ResetInMemoryInputs() {
	inMemoryInputsLock.Lock()
	defer inMemoryInputsLock.Unlock()

	for _, inp := range inMemoryInputs {
		inp.Reset()
	}
}

type InMemorySettings struct {
	Size int `cfg:"size" default:"1"`
}

type InMemoryInput struct {
	lck           sync.Mutex
	channel       chan *Message
	stopped       chan struct{}
	closedStopped bool
	settings      *InMemorySettings
}

func ProvideInMemoryInput(name string, settings *InMemorySettings) *InMemoryInput {
	inMemoryInputsLock.Lock()
	defer inMemoryInputsLock.Unlock()

	if input, ok := inMemoryInputs[name]; ok {
		return input
	}

	inMemoryInputs[name] = NewInMemoryInput(settings)

	return inMemoryInputs[name]
}

func NewInMemoryInput(settings *InMemorySettings) *InMemoryInput {
	return &InMemoryInput{
		channel:       make(chan *Message, settings.Size),
		stopped:       make(chan struct{}),
		closedStopped: false,
		settings:      settings,
	}
}

func (i *InMemoryInput) Reset() {
	i.lck.Lock()
	defer i.lck.Unlock()

	i.channel = make(chan *Message, i.settings.Size)
	i.stopped = make(chan struct{})
	i.closedStopped = false
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
	i.lck.Lock()
	defer i.lck.Unlock()

	if !i.closedStopped {
		close(i.stopped)
		i.closedStopped = true
	}
}

func (i *InMemoryInput) Data() <-chan *Message {
	return i.channel
}
