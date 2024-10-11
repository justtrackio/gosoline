package cache_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/cache"
	"github.com/karlseguin/ccache"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	assert.NotPanics(t, func() {
		cache.New[string](1, 1, 1)
	})
	assert.NotPanics(t, func() {
		cache.New[string](1, 0, 1)
	})
}

func TestNewWithConfiguration(t *testing.T) {
	assert.NotPanics(t, func() {
		cache.NewWithConfiguration[string](ccache.Configuration{}, 1)
	})
}

func TestCCache_Set_Get(t *testing.T) {
	assert.NotPanics(t, func() {
		c := cache.New[string](1, 0, time.Hour)

		c.Set("key", "value")
		value, ok := c.Get("key")

		assert.True(t, ok)
		assert.Equal(t, "value", value)
	})
}

func TestCCache_SetX_Get(t *testing.T) {
	assert.NotPanics(t, func() {
		c := cache.New[string](1, 0, 0)

		c.SetX("key", "value", time.Hour)
		value, ok := c.Get("key")

		assert.True(t, ok)
		assert.Equal(t, "value", value)
	})
}

func TestCCache_Contains(t *testing.T) {
	assert.NotPanics(t, func() {
		c := cache.New[string](1, 0, time.Hour)

		c.Set("key", "value")
		assert.Equal(t, true, c.Contains("key"))
	})
}

func TestCCache_Expire(t *testing.T) {
	assert.NotPanics(t, func() {
		c := cache.New[string](1, 0, 0)

		c.Set("key", "value")
		assert.Equal(t, false, c.Contains("key"))
	})
}

func TestCCache_Expire_Manually(t *testing.T) {
	assert.NotPanics(t, func() {
		c := cache.New[string](1, 0, time.Hour)

		c.Set("key", "value")
		c.Expire("key")
		assert.Equal(t, false, c.Contains("key"))
	})
}

func TestCCache_Expire_Manually_DoesntResurrect(t *testing.T) {
	assert.NotPanics(t, func() {
		c := cache.New[string](1, 0, time.Millisecond*100)

		c.Set("key", "value")
		// expire the item naturally - we used to have a bug where an expired item would get resurrected
		// if it was already expired (because we set the timestamp to expire it to NOW - (EXPIRE_AT - NOW)).
		// When (EXPIRE_AT - NOW) (i.e., the remaining time) is negative, the new timestamp was then in the future. Yikes!
		time.Sleep(time.Millisecond * 300)
		c.Expire("key")
		assert.Equal(t, false, c.Contains("key"))
	})
}

func TestCCache_Mutate(t *testing.T) {
	assert.NotPanics(t, func() {
		c := cache.New[string](1, 0, time.Hour)

		value := c.Mutate("key",
			func(value *string) string {
				assert.Nil(t, value)

				return "foo"
			})

		assert.Equal(t, "foo", value)

		value = c.Mutate("key",
			func(value *string) string {
				assert.NotNil(t, value)

				return *value + "bar"
			})

		assert.Equal(t, "foobar", value)
	})
}

func TestCCache_Provide(t *testing.T) {
	assert.NotPanics(t, func() {
		c := cache.New[string](1, 0, time.Hour)

		callCount := 0
		value := c.Provide("key", func() string {
			callCount++

			return "value"
		})

		assert.Equal(t, 1, callCount)
		assert.Equal(t, "value", value)

		value = c.Provide("key", func() string {
			callCount++

			return "something else"
		})

		assert.Equal(t, 1, callCount)
		assert.Equal(t, "value", value)
	})
}

func TestCCache_Provide_NotFoundNotCached(t *testing.T) {
	assert.NotPanics(t, func() {
		c := cache.New[string](1, 0, time.Hour)

		callCount := 0
		value := c.Provide("key", func() string {
			callCount++

			return ""
		})

		assert.Equal(t, 1, callCount)
		assert.Equal(t, "", value)

		value = c.Provide("key", func() string {
			callCount++

			return "something"
		})

		assert.Equal(t, 2, callCount)
		assert.Equal(t, "something", value)
	})
}

func TestCCache_Provide_NotFoundCached(t *testing.T) {
	assert.NotPanics(t, func() {
		c := cache.New[string](1, 0, time.Hour, cache.WithNotFoundTtl[string](time.Hour))

		callCount := 0
		value := c.Provide("key", func() string {
			callCount++

			return ""
		})

		assert.Equal(t, 1, callCount)
		assert.Equal(t, "", value)

		value = c.Provide("key", func() string {
			callCount++

			return "something"
		})

		assert.Equal(t, 1, callCount)
		assert.Equal(t, "", value)
	})
}

func TestCCache_ProvideWithError(t *testing.T) {
	assert.NotPanics(t, func() {
		c := cache.New[string](1, 0, time.Hour)

		callCount := 0
		value, err := c.ProvideWithError("key", func() (string, error) {
			callCount++

			return "ignored", fmt.Errorf("failed")
		})

		assert.Equal(t, 1, callCount)
		assert.EqualError(t, err, "failed")
		assert.Equal(t, "", value)

		value, err = c.ProvideWithError("key", func() (string, error) {
			callCount++

			return "value", nil
		})

		assert.Equal(t, 2, callCount)
		assert.NoError(t, err)
		assert.Equal(t, "value", value)

		value, err = c.ProvideWithError("key", func() (string, error) {
			callCount++

			return "something else", nil
		})

		assert.Equal(t, 2, callCount)
		assert.NoError(t, err)
		assert.Equal(t, "value", value)
	})
}
