package cache

import "time"
import "github.com/karlseguin/ccache"

type Cache struct {
	*ccache.Cache
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

	cache.Cache = ccache.New(config)

	return cache
}

func NewWithConfiguration(config ccache.Configuration, ttl time.Duration) *Cache {
	cache := &Cache{
		size: 0,
		ttl:  ttl,
	}
	cache.Cache = ccache.New(&config)

	return cache
}

func (c *Cache) Set(key string, value interface{}) {
	c.Cache.Set(key, value, c.ttl)
}

func (c *Cache) Contains(key string) bool {
	item := c.Get(key)

	return item != nil && !item.Expired()
}

func (c *Cache) Expire(key string) bool {
	item := c.Get(key)

	if item == nil {
		return false
	}

	remaining := item.Expires().Sub(time.Now())
	item.Extend(-1 * remaining)

	return true
}
