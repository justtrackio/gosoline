package stream

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
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

type OutputFactory func(config cfg.Config, logger log.Logger, name string) (Output, error)

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

// ensure all the types we actually write to SQS/SNS implement hasAttributes
var (
	_ hasAttributes = &Message{}
	_ hasAttributes = rawJsonMessage{}
)

func getAttributes(msg WritableMessage) map[string]interface{} {
	if withAttributes, ok := msg.(hasAttributes); ok {
		return withAttributes.GetAttributes()
	}

	return map[string]interface{}{}
}
