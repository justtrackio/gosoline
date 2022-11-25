package cache

import (
	"testing"
	"time"

	"github.com/karlseguin/ccache"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	assert.NotPanics(t, func() {
		New(1, 1, 1)
	})
	assert.NotPanics(t, func() {
		New(1, 0, 1)
	})
}

func TestNewWithConfiguration(t *testing.T) {
	assert.NotPanics(t, func() {
		NewWithConfiguration(ccache.Configuration{}, 1)
	})
}

func TestCCache_Set(t *testing.T) {
	assert.NotPanics(t, func() {
		cache := New(1, 0, 1)

		cache.Set("key", "value")
	})
}

func TestCCache_Contains(t *testing.T) {
	assert.NotPanics(t, func() {
		cache := New(1, 0, 1*time.Second)

		cache.Set("key", "value")
		assert.Equal(t, true, cache.Contains("key"))
	})
}

func TestCCache_Expire(t *testing.T) {
	assert.NotPanics(t, func() {
		cache := New(1, 0, 0)

		cache.Set("key", "value")
		assert.Equal(t, false, cache.Contains("key"))
	})
}
