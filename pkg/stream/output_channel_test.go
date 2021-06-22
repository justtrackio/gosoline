package stream_test

import (
	"github.com/applike/gosoline/pkg/log"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOutputChannel_Simple(t *testing.T) {
	logger := logMocks.NewLoggerMock()

	msg := []stream.WritableMessage{
		stream.NewMessage("hello"),
		stream.NewMessage("world"),
	}

	ch := stream.NewOutputChannel(logger, 1)
	ch.Write(msg)
	ch.Close()

	// should be able to read the message again
	readMsg, ok := ch.Read()
	assert.True(t, ok, "should be able to read message from channel")
	assert.Equal(t, msg, readMsg, "read message should match expected message")

	_, ok = ch.Read()
	assert.False(t, ok, "should not be able to read from empty channel")

	logger.AssertExpectations(t)
}

func TestOutputChannel_WriteAfterClose(t *testing.T) {
	logger := logMocks.NewLoggerMockedUntilLevel(log.PriorityWarn)

	msg := []stream.WritableMessage{
		stream.NewMessage("hello"),
		stream.NewMessage("world"),
	}

	ch := stream.NewOutputChannel(logger, 1)
	ch.Close()
	// should not crash to write after close
	ch.Write(msg)

	_, ok := ch.Read()
	assert.False(t, ok, "message written after close should be dropped")

	// should not crash to call this a second time
	ch.Close()
}
