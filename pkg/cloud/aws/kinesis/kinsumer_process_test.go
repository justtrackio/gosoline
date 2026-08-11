package kinesis

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"
)

func TestKinsumerProcessLimitsConcurrencyGlobally(t *testing.T) {
	process := &blockingProcessor{
		entered: make(chan struct{}, 4),
		release: make(chan struct{}, 4),
	}
	k := &kinsumer{
		runners: semaphore.NewWeighted(2),
	}

	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			require.NoError(t, k.process(t.Context(), t.Context(), process.Process, nil))
		})
	}

	<-process.entered
	<-process.entered
	require.Equal(t, int32(2), process.current.Load())

	for range 4 {
		process.release <- struct{}{}
	}
	wg.Wait()

	require.Equal(t, int32(2), process.maximum.Load())
}

func TestKinsumerProcessDoesNotAdmitWorkAfterCancellation(t *testing.T) {
	process := &blockingProcessor{
		entered: make(chan struct{}, 1),
		release: make(chan struct{}, 1),
	}
	k := &kinsumer{
		runners: semaphore.NewWeighted(1),
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := k.process(ctx, t.Context(), process.Process, nil)

	require.ErrorIs(t, err, errMessageProcessorStopped)
	require.Empty(t, process.entered)
}

type blockingProcessor struct {
	current atomic.Int32
	maximum atomic.Int32
	entered chan struct{}
	release chan struct{}
}

func (p *blockingProcessor) Process(_ context.Context, _ []byte) error {
	current := p.current.Add(1)
	for maximum := p.maximum.Load(); current > maximum && !p.maximum.CompareAndSwap(maximum, current); maximum = p.maximum.Load() {
	}

	p.entered <- struct{}{}
	<-p.release
	p.current.Add(-1)

	return nil
}
