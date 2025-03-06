package cache

import (
	"reflect"
	"time"

	"github.com/justtrackio/gosoline/pkg/conc"
	"github.com/karlseguin/ccache"
)

//go:generate mockery --name Cache
type Cache[T any] interface {
	// Contains checks whether any not yet expired element with key exists in the cache.
	Contains(key string) bool

	// Expire updates the expiry of the item with key if it exists.
	// Returns true if an item was expired.
	Expire(key string) bool

	// Get returns the item stored for key if it exists and is not expired.
	// Returns the item and whether a non expired item was found.
	Get(key string) (T, bool)

	// Set sets the value for key with the cache's default ttl.
	Set(key string, value T)

	// SetX sets the value for key with the provided ttl.
	SetX(key string, value T, ttl time.Duration)

	// Mutate atomically retrieves the item for key from the cache.
	// If it exists and is not expired it is passed to mutate, else a nil pointer is passed.
	// The mutated value returned by mutated is then stored in the cache
	// if it isn't its type's zero value or the [WithNotFoundTtl] option is set on cache creation.
	// Uses the cache's default ttl.
	Mutate(key string, mutate func(value *T) T) T

	// MutateX works the same as [Mutate], just uses the provided ttl instead of the cache's default.
	MutateX(key string, mutate func(value *T) T, ttl time.Duration) T

	// Provide retrieves the non expired item from the cache if it exists.
	// If not, provider is called to retrieve a new value.
	// If the provided value isn't its type's zero value or the [WithNotFoundTtl] option is
	// set on cache creation the provided value is stored in the cache.
	// Returns the retrieved or provided value.
	Provide(key string, provider func() T) T

	// ProvideWithError works the same as [Provide], but if provider returns an error the
	// provided value is immediately returned and not stored in the cache.
	ProvideWithError(key string, provider func() (T, error)) (T, error)

	// Delete deletes the item with key from the cache.
	// Returns true if the item was present, false otherwise.
	Delete(key string) bool

	// Stop stops the background worker. Operations performed on the cache after Stop
	// is called are likely to panic
	Stop()
}

type cache[T any] struct {
	base        *ccache.Cache
	ttl         time.Duration
	notFoundTtl time.Duration
	lock        conc.KeyLock
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
		lock:        conc.NewKeyLock(),
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

func (c *cache[T]) Delete(key string) bool {
	return c.base.Delete(key)
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

func (c *cache[T]) Mutate(key string, mutate func(*T) T) T {
	return c.mutate(key, mutate, c.ttl)
}

func (c *cache[T]) MutateX(key string, mutate func(*T) T, ttl time.Duration) T {
	return c.mutate(key, mutate, ttl)
}

func (c *cache[T]) mutate(key string, mutate func(*T) T, ttl time.Duration) T {
	unlock := c.lock.Lock(key)
	defer unlock()

	var result *T
	tmp, exists := c.Get(key)
	if exists {
		result = &tmp
	}

	mutated := mutate(result)

	if !isZero(mutated) {
		c.SetX(key, mutated, ttl)
	} else if c.notFoundTtl > 0 {
		// cache a typed nil-ptr. when we look up a value the next time, we will be able to cast it back to the correct type.
		c.SetX(key, mutated, c.notFoundTtl)
	}

	return mutated
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
	unlock := c.lock.Lock(key)
	defer unlock()

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

func (c *cache[T]) Stop() {
	c.base.Stop()
}
