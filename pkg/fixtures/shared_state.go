package fixtures

import (
	"context"
	"sync"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
)

type sharedStateContextKey struct{}

// SharedState keeps track of which shared fixture sets have already been loaded
// within a single test environment.
type SharedState struct {
	mutex      sync.RWMutex
	loadedKeys funk.Set[string]
}

// ProvideSharedState returns the per-environment shared fixture state.
func ProvideSharedState(ctx context.Context) (*SharedState, error) {
	return appctx.Provide(ctx, sharedStateContextKey{}, func() (*SharedState, error) {
		return &SharedState{
			loadedKeys: funk.Set[string]{},
		}, nil
	})
}

// SharedStateAppCtxValue injects an existing shared state into a fresh appctx.
func SharedStateAppCtxValue(state *SharedState) appctx.ContextValueFactory[*SharedState] {
	return func() (key any, provider func(ctx context.Context, config cfg.Config, logger log.Logger) (*SharedState, error)) {
		return sharedStateContextKey{}, func(ctx context.Context, config cfg.Config, logger log.Logger) (*SharedState, error) {
			return state, nil
		}
	}
}

func (s *SharedState) IsLoaded(key string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.loadedKeys.Contains(key)
}

func (s *SharedState) MarkLoaded(key string) {
	if key == "" {
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.loadedKeys.Add(key)
}
