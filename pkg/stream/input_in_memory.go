package stream

import (
	"context"
)

var inMemoryInputMessages = make(map[string][]*Message)

func SendToInMemoryInput(name string, message *Message) {
	if _, ok := inMemoryInputMessages[name]; !ok {
		inMemoryInputMessages[name] = make([]*Message, 0)
	}

	inMemoryInputMessages[name] = append(inMemoryInputMessages[name], message)
}

type inMemoryInput struct {
	stopped bool
	name    string
	channel chan *Message
}

func newInMemoryInput(name string) Input {
	channel := make(chan *Message)

	return &inMemoryInput{
		name:    name,
		channel: channel,
	}
}

func (i *inMemoryInput) Run(ctx context.Context) error {
	defer func() {
		close(i.channel)
	}()

	messages, ok := inMemoryInputMessages[i.name]
	if !ok {
		return nil
	}

	for _, msg := range messages {
		if i.stopped {
			break
		}

		i.channel <- msg
	}

	return nil
}

func (i *inMemoryInput) Stop() {
	i.stopped = true
}

func (i *inMemoryInput) Data() chan *Message {
	return i.channel
}
