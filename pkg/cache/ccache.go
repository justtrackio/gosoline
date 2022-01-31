package cache

import (
	"time"

	"github.com/karlseguin/ccache"
)

//go:generate mockery --name Cache
type Cache interface {
	Contains(key string) bool
	Expire(key string) bool
	Get(key string) (interface{}, bool)
	Set(key string, value interface{})
	SetX(key string, value interface{}, ttl time.Duration)
}

type cache struct {
	base *ccache.Cache
	size int
	ttl  time.Duration
}

func New(maxSize int64, pruneCount uint32, ttl time.Duration) Cache {
	cache := &cache{
		size: 0,
		ttl:  ttl,
	}

	config := ccache.Configure()
	config.MaxSize(maxSize)
	config.ItemsToPrune(pruneCount)

	cache.base = ccache.New(config)

	return cache
}

func NewWithConfiguration(config ccache.Configuration, ttl time.Duration) Cache {
	cache := &cache{
		size: 0,
		ttl:  ttl,
	}
	cache.base = ccache.New(&config)

	return cache
}

func (c *cache) Set(key string, value interface{}) {
	c.base.Set(key, value, c.ttl)
}

func (c *cache) SetX(key string, value interface{}, ttl time.Duration) {
	c.base.Set(key, value, ttl)
}

func (c *cache) Get(key string) (interface{}, bool) {
	item := c.base.Get(key)

	if item == nil {
		return nil, false
	}

	if item.Expired() {
		return nil, false
	}

	return item.Value(), true
}

func (c *cache) Contains(key string) bool {
	_, ok := c.Get(key)

	return ok
}

func (c *cache) Expire(key string) bool {
	item := c.base.Get(key)

	if item == nil {
		return false
	}

	remaining := time.Until(item.Expires())
	item.Extend(-1 * remaining)

	return true
}
