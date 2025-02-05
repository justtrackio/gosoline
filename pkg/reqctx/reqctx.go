// Package reqctx provides a [context.Context] based mechanism to cache data for the duration of a request.
// A new cache is automatically created for every request handled by the httpserver or every message consumed
// by the stream package.
package reqctx

import (
	"context"
	"sync"
)

type reqCtx struct {
	lck    sync.Mutex
	values map[any]any
}

type reqCtxKey struct{}

// New creates a new request context to hold values for the duration of a single request. The returned context is thread safe.
func New(ctx context.Context) context.Context {
	return context.WithValue(ctx, reqCtxKey{}, &reqCtx{
		values: make(map[any]any),
	})
}

// Get returns the stored value for the current request or nil if no value has yet been stored.
// Nil is also returned if the context was not wrapped using New.
// The returned value is located based on the requested type (thus, you can't store multiple values of the same type in one request).
func Get[T any](ctx context.Context) *T {
	var key *T // use a pointer as that is comparable

	reqCtx, ok := ctx.Value(reqCtxKey{}).(*reqCtx)
	if !ok {
		return nil
	}

	reqCtx.lck.Lock()
	valueI, ok := reqCtx.values[key]
	reqCtx.lck.Unlock()
	if !ok {
		return nil
	}

	value, ok := valueI.(T)
	if !ok {
		return nil
	}

	return &value
}

// Set stores or updates the stored value. It does nothing if the context was not wrapped using New.
// The value is located based on the type T, existing values in that request are overwritten.
func Set[T any](ctx context.Context, value T) {
	var key *T // use a pointer as that is comparable

	reqCtx, ok := ctx.Value(reqCtxKey{}).(*reqCtx)
	if !ok {
		return
	}

	reqCtx.lck.Lock()
	reqCtx.values[key] = value
	reqCtx.lck.Unlock()
}

// Delete removes the stored value. It does nothing if the context was not wrapped using New or no value of the given type was stored.
func Delete[T any](ctx context.Context) {
	var key *T // use a pointer as that is comparable

	reqCtx, ok := ctx.Value(reqCtxKey{}).(*reqCtx)
	if !ok {
		return
	}

	reqCtx.lck.Lock()
	delete(reqCtx.values, key)
	reqCtx.lck.Unlock()
}
