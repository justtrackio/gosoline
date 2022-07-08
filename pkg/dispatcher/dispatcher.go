package dispatcher

import (
	"context"
	"sync"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type (
	dispatcherCtxKey string

	Callback func(ctx context.Context, event interface{}) error
	Result   map[string]error
)

type Dispatcher interface {
	Fire(ctx context.Context, name string, event interface{}) Result
	On(name string, call Callback)
}

type dispatcher struct {
	logger    log.Logger
	mx        sync.RWMutex
	listeners map[string][]Callback
}

func ProvideDispatcher(ctx context.Context, _ cfg.Config, logger log.Logger) (Dispatcher, error) {
	return appctx.Provide(ctx, dispatcherCtxKey("Dispatcher"), func() (Dispatcher, error) {
		return newDispatcher(logger)
	})
}

func newDispatcher(logger log.Logger) (Dispatcher, error) {
	return &dispatcher{
		logger:    logger.WithChannel("dispatcher"),
		mx:        sync.RWMutex{},
		listeners: map[string][]Callback{},
	}, nil
}

func (d *dispatcher) Fire(ctx context.Context, name string, event interface{}) Result {
	d.mx.RLock()
	defer d.mx.RUnlock()

	callbacks, ok := d.listeners[name]
	if !ok {
		return map[string]error{}
	}

	errors := make(Result)

	for _, c := range callbacks {
		errors[name] = c(ctx, event)
	}

	return errors
}

func (d *dispatcher) On(name string, call Callback) {
	d.mx.Lock()
	defer d.mx.Unlock()

	if _, ok := d.listeners[name]; !ok {
		d.listeners[name] = make([]Callback, 0)
	}

	d.listeners[name] = append(d.listeners[name], call)
}
