package stream

import (
	"sync"

	"github.com/justtrackio/gosoline/pkg/log"
)

type OutputChannel interface {
	Read() ([]WritableMessage, bool)
	Write(msg []WritableMessage)
	Close()
	IsClosed() bool
}

type outputChannel struct {
	logger log.Logger
	ch     chan []WritableMessage
	closed bool
	lck    sync.RWMutex
}

func NewOutputChannel(logger log.Logger, bufferSize int) OutputChannel {
	return &outputChannel{
		logger: logger,
		ch:     make(chan []WritableMessage, bufferSize),
	}
}

func (c *outputChannel) Read() ([]WritableMessage, bool) {
	msg, ok := <-c.ch

	return msg, ok
}

func (c *outputChannel) Write(msg []WritableMessage) {
	c.lck.RLock()
	defer c.lck.RUnlock()

	if c.closed {
		// this can happen if we still get some traffic while everything is already shutting down.
		// this is okay as far as the producer daemon is concerned, if your data can't handle this,
		// you can't use the producer daemon anyway
		c.logger.Warn("dropped batch of %d messages: channel is already closed", len(msg))

		return
	}

	c.ch <- msg
}

func (c *outputChannel) Close() {
	c.lck.Lock()
	defer c.lck.Unlock()

	if !c.closed {
		c.closed = true
		close(c.ch)
	} else {
		c.logger.Warn("duplicate close to output channel: channel is already closed")
	}
}

func (c *outputChannel) IsClosed() bool {
	c.lck.RLock()
	defer c.lck.RUnlock()

	return c.closed
}
