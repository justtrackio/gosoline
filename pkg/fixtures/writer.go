package fixtures

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"sync"
)

type FixtureWriter interface {
	WriteFixtures(fixture *FixtureSet) error
}

type FixtureWriterFactory func(config cfg.Config, logger mon.Logger) FixtureWriter

var cachedWriters cachedWriterFactory

type cachedWriterFactory struct {
	cache map[string]FixtureWriter
	lock  sync.Mutex
}

func (c *cachedWriterFactory) New(name string, factory func() FixtureWriter) FixtureWriter {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.cache == nil {
		c.cache = make(map[string]FixtureWriter)
	}

	if _, ok := c.cache[name]; !ok {
		c.cache[name] = factory()
	}

	return c.cache[name]
}
