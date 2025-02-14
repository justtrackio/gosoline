package metric

import (
	"context"
	"sync"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/log"
)

type metricChanneCtxKey string

func providerMetricChannel(ctx context.Context) *metricChannel {
	ch, _ := appctx.Provide(ctx, metricChanneCtxKey("default"), func() (*metricChannel, error) {
		return &metricChannel{
			c: make(chan Data, 100),
		}, nil
	})

	return ch
}

type metricChannel struct {
	lck     sync.RWMutex
	logger  log.Logger
	c       chan Data
	once    sync.Once
	enabled bool
	closed  bool
}

func (c *metricChannel) write(batch Data) {
	c.lck.RLock()
	defer c.lck.RUnlock()

	if !c.enabled {
		return
	}

	if c.closed {
		c.once.Do(func() {
			c.logger.Warn("refusing to write %d metric datums to metric channel: channel is closed", len(batch))
		})
		return
	}

	c.c <- batch
}

// Lock the channel metadata, close the channel and unlock it again.
// Why do we need a RW lock for the channel? Multiple possible choices:
//   - Just read until we get nothing more - does not work if a producer
//     writes more messages after we read "everything" to the channel. If
//     the producer writes enough messages, it could actually get stuck
//     because there is no consumer left and we only buffer 100 items
//   - Just add an (atomic) boolean flag: If we check whether we closed the
//     channel and then write to it, if not, we have a time-of-check to
//     time-of-use race condition. Between our check and writing to the
//     channel someone could have closed the channel.
//   - Just use recover when you get a panic: Would work, but this is really
//     not pretty.
func (c *metricChannel) close() {
	c.lck.Lock()
	defer c.lck.Unlock()

	close(c.c)
	c.closed = true
}
