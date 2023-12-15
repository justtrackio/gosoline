package aws

import (
	"math"
	"math/rand"
	"sync"
	"time"
)

type BackoffDelayer struct {
	rand                *rand.Rand
	randLck             sync.Mutex
	initialInterval     time.Duration
	maxInterval         time.Duration
	multiplier          float64
	randomizationFactor float64
}

func NewBackoffDelayer(initialInterval time.Duration, maxInterval time.Duration) *BackoffDelayer {
	randSource := rand.NewSource(time.Now().UnixNano())

	return &BackoffDelayer{
		rand:                rand.New(randSource),
		randLck:             sync.Mutex{},
		initialInterval:     initialInterval,
		maxInterval:         maxInterval,
		multiplier:          1.5,
		randomizationFactor: 0.5,
	}
}

func (d *BackoffDelayer) BackoffDelay(attempt int, _ error) (time.Duration, error) {
	currentInterval := time.Duration(float64(d.initialInterval) * math.Pow(d.multiplier, float64(attempt-1)))
	currentInterval = d.getRandomValueFromInterval(currentInterval)

	if currentInterval > d.maxInterval {
		currentInterval = d.maxInterval
	}

	if currentInterval < 0 {
		currentInterval = 0
	}

	return currentInterval, nil
}

// Returns a random value from the following interval:
//
//	[randomizationFactor * currentInterval, randomizationFactor * currentInterval].
func (d *BackoffDelayer) getRandomValueFromInterval(currentInterval time.Duration) time.Duration {
	delta := d.randomizationFactor * float64(currentInterval)
	minInterval := float64(currentInterval) - delta
	maxInterval := float64(currentInterval) + delta

	// our random source is not thread-safe, so ensure we don't cause a race
	d.randLck.Lock()
	defer d.randLck.Unlock()

	// Get a random value from the range [minInterval, maxInterval].
	// The formula used below has a +1 because if the minInterval is 1 and the maxInterval is 3 then
	// we want a 33% chance for selecting either 1, 2 or 3.
	random := d.rand.Float64()

	return time.Duration(minInterval + (random * (maxInterval - minInterval + 1)))
}
