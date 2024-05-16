package appctx

import (
	"context"
	"fmt"
	"sync"

	"github.com/justtrackio/gosoline/pkg/conc"
)

type containerKeyType int

var containerKey containerKeyType = 1

type ErrNoApplicationContainerFound struct{}

func (e ErrNoApplicationContainerFound) Error() string {
	return "no application container found in context"
}

type ErrNoItemFound struct {
	Key interface{}
}

func (e ErrNoItemFound) Error() string {
	return fmt.Sprintf("no item with key %v found", e.Key)
}

type container struct {
	items sync.Map
	lock  conc.KeyLock
}

// WithContainer injects a thread safe pointer backed container into the provided context.
// This container is then addressable by [Provide] and [Get].
func WithContainer(ctx context.Context) context.Context {
	return context.WithValue(ctx, containerKey, &container{
		items: sync.Map{},
		lock:  conc.NewKeyLock(),
	})
}

// Provide retrieves the value behind key from the container which was injected into ctx by
// [WithContainer]. If key is not present in the container the factory will create a new value and
// store it in the container.
// This value is then accessible from all other points in a program which have access to the context
// containing the container.
// If no new value should be created when none is found for key, use [Get].
// Returns [ErrNoApplicationContainerFound] when no container is present in ctx.
func Provide[T any](ctx context.Context, key any, factory func() (T, error)) (T, error) {
	var ok bool
	var err error
	var contI, val interface{}

	if contI = ctx.Value(containerKey); contI == nil {
		return *new(T), &ErrNoApplicationContainerFound{}
	}

	cont := contI.(*container)

	unlock := cont.lock.Lock(key)
	defer unlock()

	if val, ok = cont.items.Load(key); ok {
		return val.(T), nil
	}

	if val, err = factory(); err != nil {
		return *new(T), err
	}

	cont.items.Store(key, val)

	return val.(T), nil
}

// Get retrieves the value behind key from the container which was injected into ctx by
// [WithContainer].
// If a new value should be created when none is found for key, use [Provide].
// Returns [ErrNoItemFound] if key is not present in the container.
// Returns [ErrNoApplicationContainerFound] when no container is present in ctx.
func Get[T any](ctx context.Context, key any) (T, error) {
	var ok bool
	var contI, val any

	if contI = ctx.Value(containerKey); contI == nil {
		return *new(T), &ErrNoApplicationContainerFound{}
	}

	cont := contI.(*container)

	unlock := cont.lock.Lock(key)
	defer unlock()

	if val, ok = cont.items.Load(key); ok {
		return val.(T), nil
	}

	return *new(T), &ErrNoItemFound{}
}
