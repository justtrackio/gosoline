package clock

import (
	"fmt"
	"sync"
	"time"
)

// A Ticker is similar to a Timer, but it sends the current time continuously to the channel returned by Chan.
//
//go:generate go run github.com/vektra/mockery/v2 --name Ticker
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
	lck     sync.Mutex
	wg      sync.WaitGroup
	ticker  *time.Ticker
	output  chan time.Time
	stopped chan struct{}
}

// NewRealTicker creates a new Ticker based on the current system time. Use Clock.NewTicker instead if you need to replace
// the ticker with a fake ticker for unit and integration tests.
func NewRealTicker(d time.Duration) Ticker {
	if d <= 0 {
		panic(fmt.Errorf("non-positive interval (%v) for NewTicker", d))
	}

	t := &realTicker{
		ticker:  time.NewTicker(d),
		stopped: make(chan struct{}),
		output:  make(chan time.Time, 1),
	}
	t.wg.Add(1)
	go t.transformTicks(t.stopped)

	return t
}

func (t *realTicker) Chan() <-chan time.Time {
	return t.output
}

func (t *realTicker) Reset(d time.Duration) {
	if d <= 0 {
		panic(fmt.Errorf("non-positive interval (%v) for Reset", d))
	}

	t.lck.Lock()
	defer t.lck.Unlock()

	t.stopTransformerAndWait()

	// drain any pending tick from the channel before resetting
	select {
	case <-t.output:
	default:
	}

	t.ticker.Reset(d)
	newStopped := make(chan struct{})
	t.stopped = newStopped
	t.wg.Add(1)
	go t.transformTicks(newStopped)
}

func (t *realTicker) Stop() {
	t.lck.Lock()
	defer t.lck.Unlock()

	t.stopTransformerAndWait()
	t.ticker.Stop()
}

func (t *realTicker) stopTransformerAndWait() {
	if t.stopped != nil {
		close(t.stopped)
		t.stopped = nil
		t.wg.Wait()
	}
}

func (t *realTicker) transformTicks(stopped <-chan struct{}) {
	defer t.wg.Done()

	for {
		select {
		case <-stopped:
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
