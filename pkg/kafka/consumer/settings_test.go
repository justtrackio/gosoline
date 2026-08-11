package consumer_test

import (
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kafka/consumer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSettingsConsumeDelayBelowRebalanceTimeout guards the validation of the consume delay against the rebalance
// timeout. Waiting out the delay blocks rebalances, so a delay which does not fit into the rebalance timeout gets the
// consumer kicked out of its group and silently causes duplicate message processing. Failing at startup instead.
func TestSettingsConsumeDelayBelowRebalanceTimeout(t *testing.T) {
	tests := map[string]struct {
		consumeDelay     string
		rebalanceTimeout string
		expected         time.Duration
		expectValid      bool
	}{
		"disabled":                    {consumeDelay: "0", rebalanceTimeout: "60s", expected: 0, expectValid: true},
		"below rebalance timeout":     {consumeDelay: "30s", rebalanceTimeout: "60s", expected: 30 * time.Second, expectValid: true},
		"equal to rebalance timeout":  {consumeDelay: "60s", rebalanceTimeout: "60s", expectValid: false},
		"above rebalance timeout":     {consumeDelay: "90s", rebalanceTimeout: "60s", expectValid: false},
		"raised rebalance timeout":    {consumeDelay: "90s", rebalanceTimeout: "120s", expected: 90 * time.Second, expectValid: true},
		"negative delay is no escape": {consumeDelay: "-1s", rebalanceTimeout: "60s", expectValid: false},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			settings, err := unmarshalConsumerSettings(t, map[string]any{
				"topic_id":          "testEvent",
				"consume_delay":     test.consumeDelay,
				"rebalance_timeout": test.rebalanceTimeout,
			})

			if !test.expectValid {
				assert.Error(t, err, "consume_delay %s with rebalance_timeout %s must be rejected", test.consumeDelay, test.rebalanceTimeout)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expected, settings.ConsumeDelay)
		})
	}
}

func TestSettingsConsumeDelayDefaultsToDisabled(t *testing.T) {
	settings, err := unmarshalConsumerSettings(t, map[string]any{
		"topic_id": "testEvent",
	})

	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), settings.ConsumeDelay)
}

func unmarshalConsumerSettings(t *testing.T, values map[string]any) (*consumer.Settings, error) {
	config := cfg.New()
	require.NoError(t, config.Option(cfg.WithConfigMap(map[string]any{
		"app": map[string]any{
			"name": "test-app",
			"env":  "test",
		},
		"stream.input.test": values,
	})))

	settings := &consumer.Settings{}

	return settings, config.UnmarshalKey("stream.input.test", settings)
}
