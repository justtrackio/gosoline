package stream

import "context"

var inMemoryOutputs = make(map[string]*InMemoryOutput)

type InMemoryOutput struct {
	messages []*Message
}

func ProvideInMemoryOutput(name string) *InMemoryOutput {
	if output, ok := inMemoryOutputs[name]; ok {
		return output
	}

	inMemoryOutputs[name] = &InMemoryOutput{
		messages: make([]*Message, 0),
	}

	return inMemoryOutputs[name]
}

func (o *InMemoryOutput) Len() int {
	return len(o.messages)
}

func (o *InMemoryOutput) Get(i int) (*Message, bool) {
	if len(o.messages) < i {
		return nil, false
	}

	return o.messages[i], true
}

func (o *InMemoryOutput) WriteOne(ctx context.Context, msg *Message) error {
	return o.Write(ctx, []*Message{msg})
}

func (o *InMemoryOutput) Write(_ context.Context, batch []*Message) error {
	o.messages = append(o.messages, batch...)
	return nil
}
