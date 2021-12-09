package clock

import (
	"time"
)

// A Timer will send the current time to a channel after a delay elapsed.
//go:generate mockery --name Timer
type Timer interface {
	// Chan returns the channel to which the current time will be sent once the Timer expires.
	//
	// If you enabled UTC, the timestamps are converted to UTC before you receive them.
	Chan() <-chan time.Time
	// Reset changes the timer to expire after duration d.
	//
	// For a Timer created with Clock.NewTimer, Reset should be invoked only on stopped or expired timers with drained channels.
	//
	// If a program has already received a value from t.Chan(), the timer is known to have expired and the channel drained, so
	// t.Reset can be used directly. If a program has not yet received a value from t.Chan(), however, the timer must be
	// stopped and—if Stop reports that the timer expired before being stopped—the channel explicitly drained:
	//
	// 	if !t.Stop() {
	// 		<-t.Chan()
	// 	}
	// 	t.Reset(d)
	//
	// This should not be done concurrent to other receives from the Timer's channel.
	//
	// Note that, unlike the time.Timer's Reset method, this one doesn't return a value as it is not possible to use it correctly.
	// If you try to drain the channel after calling Reset, the timer might already expire. Thus, you would drain the value
	// you would afterwards be waiting for. Reset should always be invoked on stopped or expired channels, as described above.
	//
	// A call to Reset is not thread safe. You should only ever have a single go routine responsible for a Timer or have
	// some way to coordinate between different go routines.
	//
	// If you did not drain the channel after resetting a timer and that timer expires a second time, the first value will
	// be dropped and only the second value will be available in the channel for reading.
	Reset(d time.Duration)
	// Stop prevents the Timer from firing. It returns true if the call stops the timer, false if the timer has already
	// expired or been stopped. Stop does not close the channel, to prevent a read from the channel succeeding
	// incorrectly.
	//
	// To ensure the channel is empty after a call to Stop, check the return value and drain the channel.
	// For example, assuming the program has not received from t.Chan() already:
	//
	// 	if !t.Stop() {
	// 		<-t.Chan()
	// 	}
	//
	// This cannot be done concurrent to other receives from the Timer's channel or other calls to the Timer's Stop method.
	//
	// A call to Stop is not thread safe. You should only ever have a single go routine responsible for cleaning up a Timer
	// or have some way to coordinate between different go routines.
	Stop() bool
}

type realTimer struct {
	timer  *time.Timer
	done   chan struct{}
	output chan time.Time
}

// NewRealTimer creates a new Timer based on the current system time. Use Clock.NewTimer instead if you need to replace
// the timer with a fake timer for unit and integration tests.
func NewRealTimer(d time.Duration) Timer {
	if d <= 0 {
		// ugly special case - we promised that reading from a timer with a duration of 0 would not block, but the time.Timer
		// can actually take a moment for this to be true. Thus, we create a timer with a long duration and stop it immediately
		// (so we can later reset this timer) and directly produce the correct output
		timer := &realTimer{
			timer:  time.NewTimer(time.Hour * 24 * 365),
			output: make(chan time.Time, 1),
		}
		timer.timer.Stop()
		timer.sendTick(time.Now())

		return timer
	}

	timer := &realTimer{
		timer: time.NewTimer(d),
		// use a small buffered channel so our go routine can terminate as soon as the timeout expires even if there
		// is no one receiving the time (anymore)
		output: make(chan time.Time, 1),
	}
	timer.start()

	return timer
}

func (t *realTimer) Chan() <-chan time.Time {
	return t.output
}

func (t *realTimer) Stop() bool {
	if t.done != nil {
		close(t.done)
	}
	t.done = nil

	return t.timer.Stop()
}

func (t *realTimer) Reset(d time.Duration) {
	// stop the old go routine first (if still running)
	t.Stop()
	if d <= 0 {
		t.sendTick(time.Now())

		return
	}
	// reset the timer, so we get a new tick
	t.timer.Reset(d)
	// start the go routine again for one tick
	t.start()
}

func (t *realTimer) start() {
	t.done = make(chan struct{})

	go func() {
		select {
		case <-t.done:
			return
		case tick := <-t.timer.C:
			t.sendTick(tick)
		}
	}()
}

func (t *realTimer) sendTick(tick time.Time) {
	if shouldUseUTC() {
		tick = tick.UTC()
	}

	// at this point, we either have a value already in the channel (the channel was not drained after calling Stop)
	// or an empty channel. Let us drain the channel here to ensure we can always terminate successfully even if
	// the timer is not 100% correctly used
	select {
	case <-t.output:
	default:
	}

	// we can now safely write to the 1-element buffered channel and always return immediately after
	t.output <- tick
}
