package stream

import (
	"context"
)

type WritableMessage interface {
	MarshalToBytes() ([]byte, error)
	MarshalToString() (string, error)
}

//go:generate mockery -name Output
type Output interface {
	WriteOne(ctx context.Context, msg WritableMessage) error
	Write(ctx context.Context, batch []WritableMessage) error
}

func MessagesToWritableMessages(batch []*Message) []WritableMessage {
	writableBatch := make([]WritableMessage, len(batch))

	for i, record := range batch {
		writableBatch[i] = record
	}

	return writableBatch
}

type hasAttributes interface {
	GetAttributes() map[string]interface{}
}

func getAttributes(msg WritableMessage) map[string]interface{} {
	if withAttributes, ok := msg.(hasAttributes); ok {
		return withAttributes.GetAttributes()
	}

	return map[string]interface{}{}
}
