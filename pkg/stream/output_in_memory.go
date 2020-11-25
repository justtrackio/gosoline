package stream

import (
	"context"
	"sync"
)

var inMemoryOutputsLock sync.Mutex
var inMemoryOutputs = make(map[string]*InMemoryOutput)

type InMemoryOutput struct {
	messages []*Message
}

func ProvideInMemoryOutput(name string) *InMemoryOutput {
	inMemoryOutputsLock.Lock()
	defer inMemoryOutputsLock.Unlock()

	if output, ok := inMemoryOutputs[name]; ok {
		return output
	}

	inMemoryOutputs[name] = NewInMemoryOutput()

	return inMemoryOutputs[name]
}

func NewInMemoryOutput() *InMemoryOutput {
	return &InMemoryOutput{
		messages: make([]*Message, 0),
	}
}

func (o *InMemoryOutput) Len() int {
	return len(o.messages)
}

func (o *InMemoryOutput) Get(i int) (*Message, bool) {
	if len(o.messages) <= i {
		return nil, false
	}

	return o.messages[i], true
}

func (o *InMemoryOutput) Clear() {
	o.messages = make([]*Message, 0)
}

func (o *InMemoryOutput) WriteOne(ctx context.Context, msg WritableMessage) error {
	return o.Write(ctx, []WritableMessage{msg})
}

func (o *InMemoryOutput) Write(_ context.Context, batch []WritableMessage) error {
	for _, msg := range batch {
		if streamMsg, ok := msg.(*Message); ok {
			o.messages = append(o.messages, streamMsg)

			continue
		}

		body, err := msg.MarshalToString()

		if err != nil {
			return err
		}

		o.messages = append(o.messages, &Message{
			Attributes: getAttributes(msg),
			Body:       body,
		})
	}
	return nil
}

func (o *InMemoryOutput) Size() int {
	return len(o.messages)
}

func (o *InMemoryOutput) ContainsBody(body string) bool {
	for _, msg := range o.messages {
		if msg.Body == body {
			return true
		}
	}

	return false
}
