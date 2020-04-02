package stream

import (
	"context"
)

var inMemoryInputs = make(map[string]*InMemoryInput)

type InMemorySettings struct {
	Size int `cfg:"size" default:"1"`
}

type InMemoryInput struct {
	channel chan *Message
}

func ProvideInMemoryInput(name string, settings *InMemorySettings) *InMemoryInput {
	if input, ok := inMemoryInputs[name]; ok {
		return input
	}

	inMemoryInputs[name] = &InMemoryInput{
		channel: make(chan *Message, settings.Size),
	}

	return inMemoryInputs[name]
}

func (i *InMemoryInput) Publish(messages ...*Message) {
	for _, msg := range messages {
		i.channel <- msg
	}
}

func (i *InMemoryInput) Run(_ context.Context) error {
	return nil
}

func (i *InMemoryInput) Stop() {
	close(i.channel)
}

func (i *InMemoryInput) Data() chan *Message {
	return i.channel
}
