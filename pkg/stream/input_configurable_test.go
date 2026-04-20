package stream_test

import (
	"context"
	"sync"
	"testing"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProvideConfigurableInput_ConcurrentCallsReturnSameInstance verifies that
// ProvideConfigurableInput is safe to call from multiple goroutines simultaneously
// and always returns the same instance for the same name.
func TestProvideConfigurableInput_ConcurrentCallsReturnSameInstance(t *testing.T) {
	config := cfg.New(map[string]any{
		"stream": map[string]any{
			"input": map[string]any{
				"concurrent-test": map[string]any{
					"type": "inMemory",
				},
			},
		},
	})
	logger := log.NewCliLogger()
	ctx := appctx.WithContainer(context.Background())

	const goroutines = 10
	results := make([]stream.Input, goroutines)

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			inp, err := stream.ProvideConfigurableInput(ctx, config, logger, "concurrent-test")
			require.NoError(t, err)
			results[i] = inp
		}()
	}

	wg.Wait()

	for i := 1; i < goroutines; i++ {
		assert.Equal(t, results[0], results[i], "all goroutines must receive the same instance")
	}
}
