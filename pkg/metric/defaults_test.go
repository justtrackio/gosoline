package metric

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMetricDefaults_ConcurrentReadWrite exercises addMetricDefaults (write lock)
// and amendFromDefault (read lock) concurrently. The race detector catches any data
// race, and a deadlock would cause the test to time out.
func TestMetricDefaults_ConcurrentReadWrite(t *testing.T) {
	datum := &Datum{
		MetricName: "test-concurrent-defaults",
		Priority:   PriorityHigh,
		Unit:       UnitCount,
		Kind:       KindGauge.Build(),
	}

	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(2)

		go func() {
			defer wg.Done()
			addMetricDefaults(datum)
		}()

		go func() {
			defer wg.Done()
			cpy := *datum
			amendFromDefault(&cpy)
		}()
	}

	wg.Wait()

	// Verify the default was actually registered.
	metricDefaultsLock.RLock()
	_, ok := metricDefaults[datum.Id()]
	metricDefaultsLock.RUnlock()

	assert.True(t, ok, "metric default must be registered after concurrent writes")
}

// TestAmendFromDefault_PopulatesEmptyFields verifies that amendFromDefault fills
// in missing Priority, Unit, and Kind from the registered default.
func TestAmendFromDefault_PopulatesEmptyFields(t *testing.T) {
	def := &Datum{
		MetricName: "test-amend-defaults",
		Priority:   PriorityHigh,
		Unit:       UnitCount,
		Kind:       KindGauge.Build(),
	}
	addMetricDefaults(def)

	target := &Datum{MetricName: "test-amend-defaults"}
	amendFromDefault(target)

	assert.Equal(t, PriorityHigh, target.Priority)
	assert.Equal(t, UnitCount, target.Unit)
	assert.Equal(t, KindGauge.Build(), target.Kind)
}

// TestAmendFromDefault_DoesNotOverwriteExistingFields verifies that non-zero
// fields on the incoming datum are preserved and not replaced by defaults.
func TestAmendFromDefault_DoesNotOverwriteExistingFields(t *testing.T) {
	def := &Datum{
		MetricName: "test-amend-no-overwrite",
		Priority:   PriorityHigh,
		Unit:       UnitCount,
		Kind:       KindGauge.Build(),
	}
	addMetricDefaults(def)

	target := &Datum{
		MetricName: "test-amend-no-overwrite",
		Priority:   PriorityLow,
		Unit:       UnitMilliseconds,
	}
	amendFromDefault(target)

	// Fields already set must not be overwritten.
	assert.Equal(t, PriorityLow, target.Priority)
	assert.Equal(t, UnitMilliseconds, target.Unit)
}
