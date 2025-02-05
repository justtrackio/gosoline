package guard

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/reqctx"
	"github.com/selm0/ladon"
)

type CachedManager struct {
	manager Manager
}

type managerCache struct {
	policyCache   map[string]ladon.Policy
	policiesCache map[string]ladon.Policies
}

func NewCachedManagerWithInterfaces(manager Manager) Manager {
	return &CachedManager{
		manager: manager,
	}
}

func (m CachedManager) Create(ctx context.Context, pol ladon.Policy) error {
	reqctx.Delete[managerCache](ctx)

	return m.manager.Create(ctx, pol)
}

func (m CachedManager) Update(ctx context.Context, pol ladon.Policy) error {
	reqctx.Delete[managerCache](ctx)

	return m.manager.Update(ctx, pol)
}

func (m CachedManager) Get(ctx context.Context, id string) (ladon.Policy, error) {
	return m.findPolicy(ctx, id, func() (ladon.Policy, error) {
		return m.manager.Get(ctx, id)
	})
}

func (m CachedManager) Delete(ctx context.Context, id string) error {
	reqctx.Delete[managerCache](ctx)

	return m.manager.Delete(ctx, id)
}

func (m CachedManager) GetAll(ctx context.Context, limit, offset int64) (ladon.Policies, error) {
	cacheKey := fmt.Sprintf("all:%d:%d", limit, offset)

	return m.findPolicies(ctx, cacheKey, func() (ladon.Policies, error) {
		return m.manager.GetAll(ctx, limit, offset)
	})
}

func (m CachedManager) FindRequestCandidates(ctx context.Context, r *ladon.Request) (ladon.Policies, error) {
	return m.FindPoliciesForSubject(ctx, r.Subject)
}

func (m CachedManager) FindPoliciesForSubject(ctx context.Context, subject string) (ladon.Policies, error) {
	cacheKey := fmt.Sprintf("subject:%s", subject)

	return m.findPolicies(ctx, cacheKey, func() (ladon.Policies, error) {
		return m.manager.FindPoliciesForSubject(ctx, subject)
	})
}

func (m CachedManager) FindPoliciesForResource(ctx context.Context, resource string) (ladon.Policies, error) {
	cacheKey := fmt.Sprintf("resource:%s", resource)

	return m.findPolicies(ctx, cacheKey, func() (ladon.Policies, error) {
		return m.manager.FindPoliciesForResource(ctx, resource)
	})
}

func (m CachedManager) findPolicy(ctx context.Context, cacheKey string, provider func() (ladon.Policy, error)) (ladon.Policy, error) {
	cache := reqctx.Get[managerCache](ctx)
	if cache != nil {
		result, ok := cache.policyCache[cacheKey]
		if ok {
			return result, nil
		}
	}

	policies, err := provider()
	if err != nil {
		return nil, err
	}

	if cache == nil {
		cache = &managerCache{
			policyCache:   make(map[string]ladon.Policy),
			policiesCache: make(map[string]ladon.Policies),
		}
	}

	cache.policyCache[cacheKey] = policies
	reqctx.Set(ctx, *cache)

	return policies, nil
}

func (m CachedManager) findPolicies(ctx context.Context, cacheKey string, provider func() (ladon.Policies, error)) (ladon.Policies, error) {
	cache := reqctx.Get[managerCache](ctx)
	if cache != nil {
		result, ok := cache.policiesCache[cacheKey]
		if ok {
			return result, nil
		}
	}

	policies, err := provider()
	if err != nil {
		return nil, err
	}

	if cache == nil {
		cache = &managerCache{
			policyCache:   make(map[string]ladon.Policy),
			policiesCache: make(map[string]ladon.Policies),
		}
	}

	cache.policiesCache[cacheKey] = policies
	reqctx.Set(ctx, *cache)

	return policies, nil
}
