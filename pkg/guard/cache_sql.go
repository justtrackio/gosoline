package guard

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/justtrackio/gosoline/pkg/kvstore"
	"github.com/ory/ladon"
)

type SqlCache interface {
	WithCache(query string, args []interface{}, provideValue func() (ladon.Policies, error)) (ladon.Policies, error)
}

type sqlCache struct {
	cache kvstore.KvStore
}

func NewCache() SqlCache {
	return &sqlCache{
		cache: kvstore.NewInMemoryKvStoreWithInterfaces(&kvstore.Settings{
			Ttl: time.Minute,
		}),
	}
}

func (k sqlCache) WithCache(query string, args []interface{}, provideValue func() (ladon.Policies, error)) (ladon.Policies, error) {
	cacheKeyParts := []string{
		query,
	}
	for _, arg := range args {
		cacheKeyParts = append(cacheKeyParts, fmt.Sprint(arg))
	}
	cacheKey := strings.Join(cacheKeyParts, "\u2063")

	result := ladon.Policies{}
	found, err := k.cache.Get(context.Background(), cacheKey, &result)
	if err != nil {
		found = false
	}

	if found {
		return result, nil
	}

	result, err = provideValue()
	if err != nil {
		return nil, err
	}

	_ = k.cache.Put(context.Background(), cacheKey, result)

	return result, nil
}
