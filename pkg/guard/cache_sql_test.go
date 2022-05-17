package guard_test

import (
	"fmt"
	"testing"

	"github.com/justtrackio/gosoline/pkg/guard"
	"github.com/ory/ladon"
	"github.com/stretchr/testify/assert"
)

func TestSqlCache_ErrorsNotCached(t *testing.T) {
	cache := guard.NewCache()

	calledCount := 0
	for i := 0; i < 10; i++ {
		result, err := cache.WithCache("query", []interface{}{"arg"}, func() (ladon.Policies, error) {
			calledCount++

			return nil, fmt.Errorf("fail")
		})
		assert.Nil(t, result)
		assert.EqualError(t, err, "fail")
		assert.Equal(t, i+1, calledCount)
	}
}

func TestSqlCache_ResultsCached(t *testing.T) {
	cache := guard.NewCache()

	calledCount := 0
	for i := 0; i < 10; i++ {
		result, err := cache.WithCache("query", []interface{}{"arg"}, func() (ladon.Policies, error) {
			calledCount++

			return ladon.Policies{nil}, nil
		})

		assert.Equal(t, ladon.Policies{nil}, result)
		assert.NoError(t, err)
		assert.Equal(t, 1, calledCount)
	}
}

func TestSqlCache_DifferentResultsCachedDifferently(t *testing.T) {
	cache := guard.NewCache()

	calledCount := 0
	for i := 0; i < 10; i++ {
		for j := 0; j < 10; j++ {
			result, err := cache.WithCache("query", []interface{}{"arg", i}, func() (ladon.Policies, error) {
				calledCount++

				return make(ladon.Policies, i), nil
			})

			assert.Equal(t, make(ladon.Policies, i), result)
			assert.NoError(t, err)
			assert.Equal(t, i+1, calledCount)
		}
	}
}
