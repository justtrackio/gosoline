package cache

import (
	"reflect"
	"time"

	"github.com/karlseguin/ccache"
)

//go:generate mockery --name Cache
type Cache[T any] interface {
	Contains(key string) bool
	Expire(key string) bool
	Get(key string) (T, bool)
	Set(key string, value T)
	SetX(key string, value T, ttl time.Duration)
	Provide(key string, provider func() T) T
	ProvideWithError(key string, provider func() (T, error)) (T, error)
}

type cache[T any] struct {
	base        *ccache.Cache
	ttl         time.Duration
	notFoundTtl time.Duration
}

type Option[T any] func(*cache[T])

func New[T any](maxSize int64, pruneCount uint32, ttl time.Duration, options ...Option[T]) Cache[T] {
	config := ccache.Configure()
	config.MaxSize(maxSize)
	config.ItemsToPrune(pruneCount)

	return NewWithConfiguration(*config, ttl, options...)
}

func NewWithConfiguration[T any](config ccache.Configuration, ttl time.Duration, options ...Option[T]) Cache[T] {
	cache := &cache[T]{
		base:        ccache.New(&config),
		ttl:         ttl,
		notFoundTtl: 0,
	}

	for _, opt := range options {
		opt(cache)
	}

	return cache
}

func WithNotFoundTtl[T any](notFoundTtl time.Duration) func(cache *cache[T]) {
	return func(cache *cache[T]) {
		cache.notFoundTtl = notFoundTtl
	}
}

func (c *cache[T]) Set(key string, value T) {
	c.base.Set(key, value, c.ttl)
}

func (c *cache[T]) SetX(key string, value T, ttl time.Duration) {
	c.base.Set(key, value, ttl)
}

func (c *cache[T]) Get(key string) (T, bool) {
	item := c.base.Get(key)

	if item == nil {
		var noResult T

		return noResult, false
	}

	if item.Expired() {
		var noResult T

		return noResult, false
	}

	value := item.Value().(T)

	return value, true
}

func (c *cache[T]) Contains(key string) bool {
	_, ok := c.Get(key)

	return ok
}

func (c *cache[T]) Expire(key string) bool {
	item := c.base.Get(key)

	if item == nil {
		return false
	}

	// extend the time until the item expires to the current time minus one second
	item.Extend(-time.Second)

	return true
}

func (c *cache[T]) Provide(key string, provider func() T) T {
	result, _ := provide(c, key, func() (T, bool) {
		return provider(), true
	})

	return result
}

func (c *cache[T]) ProvideWithError(key string, provider func() (T, error)) (T, error) {
	var err error

	result, ok := provide(c, key, func() (T, bool) {
		var innerResult T
		innerResult, err = provider()

		return innerResult, err == nil
	})

	if !ok {
		var noResult T

		return noResult, err
	}

	return result, nil
}

func provide[T any](c *cache[T], key string, provider func() (T, bool)) (T, bool) {
	if result, ok := c.Get(key); ok {
		return result, true
	}

	result, ok := provider()
	if !ok {
		return result, false
	}

	if !isZero(result) {
		c.Set(key, result)
	} else if c.notFoundTtl > 0 {
		// cache a typed nil-ptr. when we look up a value the next time, we will be able to cast it back to the correct type.
		c.SetX(key, result, c.notFoundTtl)
	}

	return result, true
}

func isZero[T any](v T) bool {
	// ideally, we would like to do
	//
	// var zero T
	// return v == zero
	//
	// However, that requires comparable and slices and functions are not comparable, but they can be nil and we would
	// like to check for that
	return reflect.ValueOf(&v).Elem().IsZero()
}
