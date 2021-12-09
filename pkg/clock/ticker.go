package clock

import (
	"errors"
	"sync/atomic"
	"time"
)

// A Ticker is similar to a Timer, but it sends the current time continuously to the channel returned by Chan.
//go:generate mockery --name Ticker
type Ticker interface {
	// Chan returns the channel to which the current time will be sent every time the Ticker expires.
	//
	// If you enabled UTC, the timestamps are converted to UTC before you receive them.
	Chan() <-chan time.Time
	// Reset stops a ticker and resets its period to the specified duration. The next tick will arrive after the new period
	// elapses. If you did Stop the Ticker before, it will be restarted.
	Reset(d time.Duration)
	// Stop turns off a ticker. After Stop, no more ticks will be sent. Stop does not close the channel, to prevent a
	// concurrent goroutine reading from the channel from seeing an erroneous "tick".
	Stop()
}

type realTicker struct {
	ticker           *time.Ticker
	output           chan time.Time
	close            chan struct{}
	runningIteration int32
}

// NewRealTicker creates a new Ticker based on the current system time. Use Clock.NewTicker instead if you need to replace
// the ticker with a fake ticker for unit and integration tests.
func NewRealTicker(d time.Duration) Ticker {
	t := &realTicker{
		ticker: time.NewTicker(d),
		close:  make(chan struct{}),
		output: make(chan time.Time),
	}
	go t.transformTicks()

	return t
}

func (t *realTicker) Chan() <-chan time.Time {
	return t.output
}

func (t *realTicker) Reset(d time.Duration) {
	if d <= 0 {
		panic(errors.New("non-positive interval for Reset"))
	}
	t.stopTransformer()
	t.ticker.Reset(d)
	go t.transformTicks()
}

func (t *realTicker) Stop() {
	t.stopTransformer()
	t.ticker.Stop()
}

func (t *realTicker) stopTransformer() {
	// tell the transformer it should stop. We need to do this, so if we send something to the close
	// channel, but the transformer doesn't see it, it will see it on the next loop iteration
	atomic.AddInt32(&t.runningIteration, 1)

	select {
	case t.close <- struct{}{}:
		// stopped the go routine transforming the timezone
		break
	default:
		// there was no go routine running right now
	}
}

func (t *realTicker) transformTicks() {
	// remember which iteration of the ticker we are. If this ever changes, we should've terminated, but missed it
	iteration := atomic.LoadInt32(&t.runningIteration)

	for iteration == atomic.LoadInt32(&t.runningIteration) {
		select {
		case <-t.close:
			return
		case tick := <-t.ticker.C:
			if shouldUseUTC() {
				tick = tick.UTC()
			}

			select {
			case t.output <- tick:
			default:
			}
		}
	}
}
