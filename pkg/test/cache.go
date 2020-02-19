package test

import (
	"sync"
)

type simpleCache struct {
	lock  sync.Mutex
	cache map[string]interface{}
}

func (c *simpleCache) New(name string, factory func() interface{}) interface{} {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.cache == nil {
		c.cache = make(map[string]interface{})
	}

	if _, ok := c.cache[name]; !ok {
		c.cache[name] = factory()
	}

	return c.cache[name]
}
