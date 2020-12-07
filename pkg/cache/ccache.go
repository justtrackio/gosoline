package cache

import "time"
import "github.com/karlseguin/ccache"

type Cache struct {
	base *ccache.Cache
	size int
	ttl  time.Duration
}

func New(maxSize int64, pruneCount uint32, ttl time.Duration) *Cache {
	cache := &Cache{
		size: 0,
		ttl:  ttl,
	}

	config := ccache.Configure()
	config.MaxSize(maxSize)
	config.ItemsToPrune(pruneCount)

	cache.base = ccache.New(config)

	return cache
}

func NewWithConfiguration(config ccache.Configuration, ttl time.Duration) *Cache {
	cache := &Cache{
		size: 0,
		ttl:  ttl,
	}
	cache.base = ccache.New(&config)

	return cache
}

func (c *Cache) Set(key string, value interface{}) {
	c.base.Set(key, value, c.ttl)
}

func (c *Cache) SetX(key string, value interface{}, ttl time.Duration) {
	c.base.Set(key, value, ttl)
}

func (c *Cache) Get(key string) (interface{}, bool) {
	item := c.base.Get(key)

	if item == nil {
		return nil, false
	}

	if item.Expired() {
		return nil, false
	}

	return item.Value(), true
}

func (c *Cache) Contains(key string) bool {
	_, ok := c.Get(key)

	return ok
}

func (c *Cache) Expire(key string) bool {
	item := c.base.Get(key)

	if item == nil {
		return false
	}

	remaining := time.Until(item.Expires())
	item.Extend(-1 * remaining)

	return true
}
