package stream

import (
	"context"
)

//go:generate mockery -name Output
type Output interface {
	WriteOne(ctx context.Context, msg *Message) error
	Write(ctx context.Context, batch []*Message) error
}
