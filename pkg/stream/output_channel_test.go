package stream_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/log"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/stretchr/testify/assert"
)

func TestOutputChannel_Simple(t *testing.T) {
	ctx := t.Context()
	logger := logMocks.NewLoggerMock(logMocks.WithTestingT(t))

	msg := []stream.WritableMessage{
		stream.NewMessage("hello"),
		stream.NewMessage("world"),
	}

	ch := stream.NewOutputChannel(logger, 1)
	ch.Write(ctx, msg)
	ch.Close(ctx)

	// should be able to read the message again
	readMsg, ok := ch.Read()
	assert.True(t, ok, "should be able to read message from channel")
	assert.Equal(t, msg, readMsg, "read message should match expected message")

	_, ok = ch.Read()
	assert.False(t, ok, "should not be able to read from empty channel")
}

func TestOutputChannel_WriteAfterClose(t *testing.T) {
	ctx := t.Context()
	logger := logMocks.NewLoggerMock(logMocks.WithMockUntilLevel(log.PriorityWarn), logMocks.WithTestingT(t))

	msg := []stream.WritableMessage{
		stream.NewMessage("hello"),
		stream.NewMessage("world"),
	}

	ch := stream.NewOutputChannel(logger, 1)
	ch.Close(ctx)
	// should not crash to write after close
	ch.Write(ctx, msg)

	_, ok := ch.Read()
	assert.False(t, ok, "message written after close should be dropped")

	// should not crash to call this a second time
	ch.Close(ctx)
}
