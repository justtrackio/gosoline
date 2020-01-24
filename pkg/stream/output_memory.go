package stream

import "context"

type OutputMemory struct {
	messages []*Message
}

func NewOutputMemory() *OutputMemory {
	return &OutputMemory{
		messages: make([]*Message, 0),
	}
}

func (o *OutputMemory) WriteOne(ctx context.Context, msg *Message) error {
	return o.Write(ctx, []*Message{msg})
}

func (o *OutputMemory) Write(ctx context.Context, batch []*Message) error {
	o.messages = append(o.messages, batch...)
	return nil
}

func (o *OutputMemory) Size() int {
	return len(o.messages)
}

func (o *OutputMemory) ContainsBody(body string) bool {
	for _, msg := range o.messages {
		if msg.Body == body {
			return true
		}
	}

	return false
}

func (o *OutputMemory) Messages() []*Message {
	return o.messages
}
