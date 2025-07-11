package coffin

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/justtrackio/gosoline/pkg/mdl"
	errorsPkg "github.com/pkg/errors"
)

// A Graveyard allows you to monitor the execution of multiple goroutines. You can wait until no goroutine is executing anymore, but a Graveyard
// is infinitely reusable as long as no goroutine panics or returns an error.
type Graveyard interface {
	// Err returns the error from the graveyard if any spawned goroutine panicked or returned an error
	Err() error
	// Ctx returns the context which would be passed to a function in GoWithContext.
	Ctx() context.Context
	// Go spawns a new goroutine and runs the given function. If the go routine panics or returns an error, Err will return it.
	Go(name string, f func() error, options ...GraveyardOption)
	// GoWithContext is the same as Go, but passes the context from the Graveyard to the function.
	GoWithContext(name string, f func(ctx context.Context) error, options ...GraveyardOption)
	// Wait waits for all spawned tasks to terminate (i.e., Running returns 0). If new goroutines are spawned after Wait returns,
	// the next call to Wait will wait again.
	Wait() error
	// Started returns the number of started goroutines.
	Started() int
	// Running returns the number of currently running goroutines.
	Running() int
	// Terminated returns the number of goroutines that have already returned.
	Terminated() int
	// Entomb returns a Coffin for the current state of the Graveyard. You can use it to wait on a channel until all goroutines are finished, or
	// kill the currently running goroutines. If you kill a Coffin, you can no longer reuse the Graveyard.
	//
	// Calling Entomb always returns the same Coffin as long as there are running goroutines (or the Graveyard is freshly created). Once all
	// goroutines inside your Graveyard finish, and you start another one with Go, a new Coffin will be returned by Entomb from that point on.
	// Thus, you should first schedule all goroutines (even if they immediately finish running) and the call Entomb.
	Entomb() Coffin
	// Kill cancels the context of the Graveyard with the given reason.
	//
	// Although Kill may be called multiple times, only the first non-nil error is recorded as the death reason.
	Kill(err error)
}

type graveyard struct {
	// copy of our context to reuse once we start returning a new Coffin
	baseCtx context.Context
	// current context and channels for the current Coffin
	ctx       context.Context
	cancelCtx context.CancelCauseFunc
	dead      chan Void
	dying     chan Void
	alive     *int32
	// bookkeeping data
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

// WithContext allows you to overwrite the context for a single goroutine as well as setting the default context for a Graveyard.
func WithContext(ctx context.Context) GraveyardOption {
	return func(options *graveyardOptions) {
		options.ctx = ctx
	}
}

// WithLabels configures the labels for a single goroutine or the default labels for a Graveyard.
func WithLabels(labels map[string]string) GraveyardOption {
	return func(options *graveyardOptions) {
		options.labels = append(options.labels, labels)
	}
}

// WithErrorWrapper ensures a panic or returned error from the spawned goroutine is wrapped using the given message and arguments.
// It should only be passed to Graveyard.Go and Graveyard.GoWithContext.
func WithErrorWrapper(msg string, args ...any) GraveyardOption {
	return func(options *graveyardOptions) {
		options.errorWrapper = func(err error) error {
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
	baseCtx := pprof.WithLabels(opts.ctx, labelSet)

	g := &graveyard{
		baseCtx: baseCtx,
	}
	g.initCoffin()

	return g
}

func (g *graveyard) Err() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	return g.err
}

func (g *graveyard) Ctx() context.Context {
	return g.initCoffin()
}

func (g *graveyard) Go(name string, f func() error, options ...GraveyardOption) {
	pkg := g.callerPackage()

	g.goWithContext(name, pkg, func(ctx context.Context) error { return f() }, options...)
}

func (g *graveyard) GoWithContext(name string, f func(ctx context.Context) error, options ...GraveyardOption) {
	pkg := g.callerPackage()

	g.goWithContext(name, pkg, f, options...)
}

// returns the package path of the function that called this function.
func (g *graveyard) callerPackage() string {
	pc, _, _, ok := runtime.Caller(2) // 1 = skip this function
	if !ok {
		return ""
	}

	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return ""
	}

	fullFuncName := fn.Name() // e.g., "github.com/user/project/pkg.MyFunc"
	lastSlash := strings.LastIndex(fullFuncName, "/")
	if lastSlash == -1 {
		lastSlash = 0
	} else {
		lastSlash++ // move past the slash
	}

	// trim to package.function
	pkgAndFunc := fullFuncName[lastSlash:]
	firstDot := strings.Index(pkgAndFunc, ".")
	if firstDot == -1 {
		return ""
	}

	// return just the package part
	return pkgAndFunc[:firstDot]
}

func (g *graveyard) goWithContext(name string, pkg string, f func(ctx context.Context) error, options ...GraveyardOption) {
	g.initCoffin()

	opts := graveyardOptions{
		ctx:          g.ctx,
		errorWrapper: defaultErrorWrapper,
	}
	for _, option := range options {
		option(&opts)
	}

	atomic.AddInt64(&g.started, 1)
	atomic.AddInt64(&g.running, 1)
	g.wg.Add(1)

	go g.runLabeled(opts.ctx, name, pkg, opts.labels, func() {
		defer g.done()
		defer func() {
			panicErr := ResolveRecovery(recover())
			if panicErr != nil {
				g.setErr(opts.errorWrapper(panicErr))
			}
		}()

		if err := f(opts.ctx); err != nil {
			g.setErr(opts.errorWrapper(err))
		}
	})
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

func (g *graveyard) Entomb() Coffin {
	g.mu.Lock()
	defer g.mu.Unlock()

	return coffin{
		Graveyard: g,
		dead:      g.dead,
		dying:     g.dying,
		alive:     g.alive,
	}
}

func (g *graveyard) initCoffin() context.Context {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.ctx != nil {
		return g.ctx
	}

	g.ctx, g.cancelCtx = context.WithCancelCause(g.baseCtx)
	g.dead = make(chan Void)
	g.dying = make(chan Void)
	g.alive = mdl.Box[int32](1)

	ctx := g.ctx
	dying := g.dying

	go g.runLabeled(ctx, "context watcher", "coffin", nil, func() {
		select {
		case <-dying:
			return
		case <-ctx.Done():
		}

		g.mu.Lock()
		defer g.mu.Unlock()

		// if we are still using the same dying channel, no new coffin was created in the meantime
		if g.dying == dying {
			g.kill(ctx.Err())
		}
	})

	return g.ctx
}

func (g *graveyard) done() {
	atomic.AddInt64(&g.running, -1)
	atomic.AddInt64(&g.terminated, 1)
	g.wg.Done()

	g.mu.Lock()
	defer g.mu.Unlock()

	if atomic.LoadInt64(&g.running) == 0 {
		// nothing is running anymore, close all channels and stop anything
		atomic.StoreInt32(g.alive, 0)

		g.closeIfOpen(g.dying)
		g.closeIfOpen(g.dead)
		g.dying = nil
		g.dead = nil
		g.ctx = nil
		g.cancelCtx = nil
	}
}

func (g *graveyard) Kill(reason error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.kill(reason)
}

func (g *graveyard) kill(reason error) {
	if g.cancelCtx != nil {
		g.cancelCtx(reason)
		g.cancelCtx = nil
	}

	g.closeIfOpen(g.dying)

	if reason != nil && g.err == nil {
		g.setErrLocked(reason)
	}
}

func (g *graveyard) closeIfOpen(c chan Void) {
	if c == nil {
		return
	}

	// close c if it is still open. As we never write to a channel, being able to read from it means it is already closed.
	// this method assumes we hold a lock and thus can't be called concurrently.
	select {
	case <-c:
	default:
		close(c)
	}
}

func (g *graveyard) setErr(err error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.setErrLocked(err)
}

func (g *graveyard) setErrLocked(err error) {
	if g.err == nil {
		g.err = err
	} else {
		g.err = errors.Join(g.err, err)
	}
}

func (g *graveyard) runLabeled(ctx context.Context, name string, pkg string, labels []map[string]string, f func()) {
	labelArgs := []string{
		"name",
		fmt.Sprintf("%s/%s", pkg, name),
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

func defaultErrorWrapper(err error) error {
	return err
}
