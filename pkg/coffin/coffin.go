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

var errCoffinDying = fmt.Errorf("coffin is dying")

// A Coffin allows you to monitor the execution of multiple goroutines. You can wait until no goroutine is executing anymore, but a Coffin
// is infinitely reusable as long as no goroutine panics or returns an error.
type Coffin interface {
	// Err returns the error from the coffin if any spawned goroutine panicked or returned an error
	Err() error
	// Ctx returns the context which would be passed to a function in GoWithContext. After all go routines in a Coffin have finished executing,
	// Ctx continues to return the old context until a new go routine is started.
	Ctx() context.Context
	// Go spawns a new goroutine and runs the given function. If the go routine panics or returns an error, Err will return it.
	Go(name string, f func() error, options ...Option)
	// TODO: make GoWithContext default and drop Go?
	// GoWithContext is the same as Go, but passes the context from the Coffin to the function.
	GoWithContext(name string, f func(ctx context.Context) error, options ...Option)
	// Wait waits for all spawned tasks to terminate (i.e., Running returns 0). If new goroutines are spawned after Wait returns,
	// the next call to Wait will wait again.
	Wait() error
	// Started returns the number of started goroutines.
	Started() int
	// Running returns the number of currently running goroutines.
	Running() int
	// Terminated returns the number of goroutines that have already returned.
	Terminated() int
	// Entomb returns a Tomb for the current state of the Coffin. You can use it to wait on a channel until all goroutines are finished, or
	// kill the currently running goroutines. If you kill a Tomb, you can no longer reuse the Coffin.
	//
	// Calling Entomb always returns the same Tomb as long as there are running goroutines (or the Coffin is freshly created). Once all
	// goroutines inside your Coffin finish, and you start another one with Go, a new Tomb will be returned by Entomb from that point on.
	// Thus, you should first schedule all goroutines (even if they immediately finish running) and the call Entomb.
	Entomb() Tomb
	// Kill cancels the context of the Coffin with the given reason.
	//
	// Although Kill may be called multiple times, only the first non-nil error is recorded as the death reason.
	Kill(err error)
}

type coffin struct {
	// copy of our context to reuse once we start returning a new Tomb
	baseCtx context.Context
	// current context and channels for the current Tomb
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

type coffinOptions struct {
	ctx          context.Context
	labels       []map[string]string
	errorWrapper func(err error, includeStackTrace bool) error
}

type Option func(options *coffinOptions)

// WithContext allows you to overwrite the context for a single goroutine as well as setting the default context for a Coffin.
func WithContext(ctx context.Context) Option {
	return func(options *coffinOptions) {
		options.ctx = ctx
	}
}

// WithLabels configures the labels for a single goroutine or the default labels for a Coffin.
func WithLabels(labels map[string]string) Option {
	return func(options *coffinOptions) {
		options.labels = append(options.labels, labels)
	}
}

// WithErrorWrapper ensures a panic or returned error from the spawned goroutine is wrapped using the given message and arguments.
// It should only be passed to Coffin.Go and Coffin.GoWithContext.
func WithErrorWrapper(msg string, args ...any) Option {
	return func(options *coffinOptions) {
		options.errorWrapper = func(err error, includeStackTrace bool) error {
			if includeStackTrace {
				return errorsPkg.Wrapf(err, msg, args...)
			}

			return fmt.Errorf("%s: %w", fmt.Sprintf(msg, args...), err)
		}
	}
}

// New returns a new coffin with the given set of default labels attached to every spawned goroutine.
func New(ctx context.Context, options ...Option) Coffin {
	opts := coffinOptions{
		ctx: ctx,
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

	g := &coffin{
		baseCtx: baseCtx,
	}
	g.resetCoffin()

	return g
}

func (g *coffin) Err() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	return g.err
}

func (g *coffin) Ctx() context.Context {
	return g.ctx
}

func (g *coffin) Go(name string, f func() error, options ...Option) {
	pkg := g.callerPackage()

	g.goWithContext(name, pkg, func(ctx context.Context) error { return f() }, options...)
}

func (g *coffin) GoWithContext(name string, f func(ctx context.Context) error, options ...Option) {
	pkg := g.callerPackage()

	g.goWithContext(name, pkg, f, options...)
}

// returns the package path of the function that called this function.
func (g *coffin) callerPackage() string {
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

func (g *coffin) goWithContext(name string, pkg string, f func(ctx context.Context) error, options ...Option) {
	g.mu.Lock()
	defer g.mu.Unlock()

	newStarted := atomic.AddInt64(&g.started, 1)
	newRunning := atomic.AddInt64(&g.running, 1)
	g.wg.Add(1)

	if newRunning == 1 && newStarted != 1 {
		// we finished running the last go routine already and are reusing the coffin.
		// we can't call this on the first call as we already init the coffin in the constructor.
		g.resetCoffin()
	}

	opts := coffinOptions{
		ctx: g.ctx,
		errorWrapper: func(err error, includeStackTrace bool) error {
			if includeStackTrace {
				return errorsPkg.Wrapf(err, "failed to execute task %q from package %q", name, pkg)
			}

			return fmt.Errorf("failed to execute task %q from package %q: %w", name, pkg, err)
		},
	}
	for _, option := range options {
		option(&opts)
	}

	go RunLabeled(opts.ctx, fmt.Sprintf("%s/%s", pkg, name), func() {
		defer g.done()
		defer func() {
			panicErr := ResolveRecovery(recover())
			if panicErr != nil {
				g.Kill(opts.errorWrapper(panicErr, true))
			}
		}()

		if err := f(opts.ctx); err != nil {
			g.Kill(opts.errorWrapper(err, false))
		}
	}, opts.labels...)
}

func (g *coffin) Wait() error {
	g.wg.Wait()

	return g.Err()
}

func (g *coffin) Started() int {
	return int(atomic.LoadInt64(&g.started))
}

func (g *coffin) Running() int {
	return int(atomic.LoadInt64(&g.running))
}

func (g *coffin) Terminated() int {
	return int(atomic.LoadInt64(&g.terminated))
}

func (g *coffin) Entomb() Tomb {
	g.mu.Lock()
	defer g.mu.Unlock()

	return tomb{
		Coffin: g,
		dead:   g.dead,
		dying:  g.dying,
		alive:  g.alive,
	}
}

func (g *coffin) resetCoffin() {
	g.ctx, g.cancelCtx = context.WithCancelCause(g.baseCtx)
	g.dead = make(chan Void)
	g.dying = make(chan Void)
	g.alive = mdl.Box[int32](1)

	if g.baseCtx.Done() != nil {
		dying := g.dying

		go RunLabeled(g.baseCtx, "coffin/context watcher", func() {
			select {
			case <-dying:
			case <-g.baseCtx.Done():
				g.Kill(nil)
			}
		})
	}
}

func (g *coffin) done() {
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
		if g.cancelCtx != nil {
			g.cancelCtx(errCoffinDying)
			g.cancelCtx = nil
		}
	}
}

func (g *coffin) Kill(reason error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.kill(reason)
}

func (g *coffin) kill(reason error) {
	if reason == nil {
		reason = errCoffinDying
	}

	if g.cancelCtx != nil {
		g.cancelCtx(reason)
		g.cancelCtx = nil
	}

	g.closeIfOpen(g.dying)

	g.setErr(reason)
}

func (g *coffin) closeIfOpen(c chan Void) {
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

func (g *coffin) setErr(err error) {
	if err == nil || errors.Is(err, errCoffinDying) {
		return
	}

	if g.err == nil {
		g.err = err
	} else {
		g.err = errors.Join(g.err, err)
	}
}

// RunLabeled calls the provided function after setting labels for the goroutine to denote the name and other attributes.
func RunLabeled(ctx context.Context, name string, f func(), labels ...map[string]string) {
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
