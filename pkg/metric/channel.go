package metric

import (
	"sync"

	"github.com/justtrackio/gosoline/pkg/log"
)

var metricChannelContainer = struct {
	sync.Mutex
	instance *metricChannel
}{}

func providerMetricChannel(configure func(channel *metricChannel)) *metricChannel {
	metricChannelContainer.Lock()
	defer metricChannelContainer.Unlock()

	if metricChannelContainer.instance != nil {
		configure(metricChannelContainer.instance)

		return metricChannelContainer.instance
	}

	metricChannelContainer.instance = &metricChannel{
		hasData: make(chan struct{}, 1),
	}
	configure(metricChannelContainer.instance)

	return metricChannelContainer.instance
}

// metricChannel is a similar implementation to a go channel, but with an unbounded
// message buffer. we need an unbounded queue because we otherwise have to deal with
// too many problems of services writing to the channel when it is already full
// (writing metric while consuming metrics or writing metrics while the channel gets
// closed). if a service writes too many metrics and blows through its memory allocation,
// this is a much louder error, causes the service to restart, and the service to heal
// automatically (to some degree at least)
type metricChannel struct {
	lck     sync.Mutex
	logger  log.Logger
	hasData chan struct{}
	data    Data
	enabled bool
	closed  bool
}

func (c *metricChannel) read() Data {
	c.lck.Lock()
	defer c.lck.Unlock()

	data := c.data
	c.data = nil

	// no need to clear hasData - the caller should have done this, but it is
	// also not an error for this flag to be wrongly set

	return data
}

func (c *metricChannel) write(batch Data) {
	c.lck.Lock()
	defer c.lck.Unlock()

	// we just return on closed channels because we can't avoid some services
	// writing some metrics when they are shut down. if we are already closed
	// at that point, we log pointless messages about the channel being closed
	// already, although there isn't really anything we can do about it
	// also: writing a warning here would be a terrible idea as this could
	// trigger a metric to be written, calling this method again (but the lock
	// is already taken)
	if !c.enabled || c.closed {
		return
	}

	c.data = append(c.data, batch...)

	// we still need to be able to read from this like from a channel... so fake
	// it with a dummy channel which can only hold a single value (we only need
	// a flag whether there could be data in here - it is okay for hasData to
	// return a value even though c.data has length 0)
	select {
	case c.hasData <- struct{}{}:
	default:
	}
}

func (c *metricChannel) close() {
	c.lck.Lock()
	defer c.lck.Unlock()

	c.closed = true
}
