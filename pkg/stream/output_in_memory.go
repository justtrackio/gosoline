package stream

import (
	"context"
	"sync"
)

var inMemoryOutputsLock sync.Mutex
var inMemoryOutputs = make(map[string]*InMemoryOutput)

func ResetInMemoryOutputs() {
	inMemoryOutputsLock.Lock()
	defer inMemoryOutputsLock.Unlock()

	for _, inp := range inMemoryOutputs {
		inp.Clear()
	}
}

type InMemoryOutput struct {
	lck      sync.Mutex
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
	o.lck.Lock()
	defer o.lck.Unlock()

	return len(o.messages)
}

func (o *InMemoryOutput) Get(i int) (*Message, bool) {
	o.lck.Lock()
	defer o.lck.Unlock()

	if len(o.messages) <= i {
		return nil, false
	}

	return o.messages[i], true
}

func (o *InMemoryOutput) Clear() {
	o.lck.Lock()
	defer o.lck.Unlock()

	o.messages = make([]*Message, 0)
}

func (o *InMemoryOutput) WriteOne(ctx context.Context, msg WritableMessage) error {
	return o.Write(ctx, []WritableMessage{msg})
}

func (o *InMemoryOutput) Write(_ context.Context, batch []WritableMessage) error {
	o.lck.Lock()
	defer o.lck.Unlock()

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
	return o.Len()
}

func (o *InMemoryOutput) ContainsBody(body string) bool {
	o.lck.Lock()
	defer o.lck.Unlock()

	for _, msg := range o.messages {
		if msg.Body == body {
			return true
		}
	}

	return false
}
