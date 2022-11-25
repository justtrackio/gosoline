package aws

import (
	"math"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type BackoffDelayer struct {
	initialInterval     time.Duration
	maxInterval         time.Duration
	multiplier          float64
	randomizationFactor float64
}

func NewBackoffDelayer(initialInterval time.Duration, maxInterval time.Duration) *BackoffDelayer {
	return &BackoffDelayer{
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

	return currentInterval, nil
}

// Returns a random value from the following interval:
//
//	[randomizationFactor * currentInterval, randomizationFactor * currentInterval].
func (d *BackoffDelayer) getRandomValueFromInterval(currentInterval time.Duration) time.Duration {
	delta := d.randomizationFactor * float64(currentInterval)
	minInterval := float64(currentInterval) - delta
	maxInterval := float64(currentInterval) + delta

	// Get a random value from the range [minInterval, maxInterval].
	// The formula used below has a +1 because if the minInterval is 1 and the maxInterval is 3 then
	// we want a 33% chance for selecting either 1, 2 or 3.
	random := rand.Float64()

	return time.Duration(minInterval + (random * (maxInterval - minInterval + 1)))
}
