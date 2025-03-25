package coffin

import (
	"context"
	"errors"
	"runtime/pprof"
	"sync"
	"sync/atomic"

	errorsPkg "github.com/pkg/errors"
)

type Graveyard interface {
	// Err returns the error from the graveyard if any spawned goroutine panicked or returned an error
	Err() error
	// Spawn spawns a new goroutine and runs the given function. If the go routine panics or returns an error, Err will return it.
	Spawn(name string, f func() error, options ...GraveyardOption)
	// SpawnWithContext is the same as Spawn, but passes the given context to the function.
	SpawnWithContext(name string, ctx context.Context, f func(ctx context.Context) error, options ...GraveyardOption)
	// Wait waits for all spawned tasks to terminate (i.e., Running returns 0). If new goroutines are spawned after Wait returns,
	// the next call to Wait will wait again.
	Wait() error
	// Started returns the number of started goroutines.
	Started() int
	// Running returns the number of currently running goroutines.
	Running() int
	// Terminated returns the number of goroutines that have already returned.
	Terminated() int
}

type graveyard struct {
	ctx        context.Context
	mu         sync.Mutex
	err        error
	wg         sync.WaitGroup
	started    int64
	running    int64
	terminated int64
}

type graveyardOptions struct {
	ctx          context.Context
	labels       []map[string]string
	errorWrapper func(err error) error
}

type GraveyardOption func(options *graveyardOptions)

func WithLabelsFromContext(ctx context.Context) GraveyardOption {
	return func(options *graveyardOptions) {
		options.ctx = ctx
	}
}

func WithLabels(labels map[string]string) GraveyardOption {
	return func(options *graveyardOptions) {
		options.labels = append(options.labels, labels)
	}
}

func WithErrorWrapper(msg string, args ...any) GraveyardOption {
	return func(options *graveyardOptions) {
		options.errorWrapper = func(err error) error {
			if err == nil {
				return nil
			}

			return errorsPkg.Wrapf(err, msg, args...)
		}
	}
}

// NewGraveyard returns a new graveyard with the given set of default labels attached to every spawned goroutine.
func NewGraveyard(options ...GraveyardOption) Graveyard {
	opts := graveyardOptions{
		ctx: context.Background(),
	}
	for _, option := range options {
		option(&opts)
	}

	var labelArgs []string
	for _, labelsMap := range opts.labels {
		for k, v := range labelsMap {
			labelArgs = append(labelArgs, k, v)
		}
	}

	labelSet := pprof.Labels(labelArgs...)
	ctx := pprof.WithLabels(opts.ctx, labelSet)

	return &graveyard{
		ctx: ctx,
	}
}

func (g *graveyard) Err() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	return g.err
}

func (g *graveyard) Spawn(name string, f func() error, options ...GraveyardOption) {
	opts := graveyardOptions{
		ctx: g.ctx,
		errorWrapper: func(err error) error {
			return err
		},
	}
	for _, option := range options {
		option(&opts)
	}

	atomic.AddInt64(&g.started, 1)
	atomic.AddInt64(&g.running, 1)
	g.wg.Add(1)

	go g.runLabeled(opts.ctx, name, opts.labels, func() {
		defer g.wg.Done()
		defer atomic.AddInt64(&g.running, -1)
		defer atomic.AddInt64(&g.terminated, 1)
		defer func() {
			panicErr := ResolveRecovery(recover())
			if panicErr != nil {
				g.setErr(opts.errorWrapper(panicErr))
			}
		}()

		if err := f(); err != nil {
			g.setErr(opts.errorWrapper(err))
		}
	})
}

func (g *graveyard) SpawnWithContext(name string, ctx context.Context, f func(ctx context.Context) error, options ...GraveyardOption) {
	g.Spawn(name, func() error { return f(ctx) }, options...)
}

func (g *graveyard) Wait() error {
	g.wg.Wait()

	return g.Err()
}

func (g *graveyard) Started() int {
	return int(atomic.LoadInt64(&g.started))
}

func (g *graveyard) Running() int {
	return int(atomic.LoadInt64(&g.running))
}

func (g *graveyard) Terminated() int {
	return int(atomic.LoadInt64(&g.terminated))
}

func (g *graveyard) setErr(err error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.err == nil {
		g.err = err
	} else {
		g.err = errors.Join(g.err, err)
	}
}

func (g *graveyard) runLabeled(ctx context.Context, name string, labels []map[string]string, f func()) {
	labelArgs := []string{
		"name",
		name,
	}

	for _, labelsMap := range labels {
		for k, v := range labelsMap {
			labelArgs = append(labelArgs, k, v)
		}
	}

	labelSet := pprof.Labels(labelArgs...)
	pprof.Do(ctx, labelSet, func(ctx context.Context) {
		f()
	})
}
