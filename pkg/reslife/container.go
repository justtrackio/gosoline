package reslife

import (
	"context"
	"fmt"
	"sync"

	"github.com/justtrackio/gosoline/pkg/appctx"
)

type (
	container struct {
		lck       sync.Mutex
		factories []LifeCycleerFactory
	}
	containerCtxKey struct{}
)

func AddLifeCycleer(ctx context.Context, factory LifeCycleerFactory) error {
	var err error
	var cont *container

	if cont, err = provideContainer(ctx); err != nil {
		return fmt.Errorf("could not add lifeCycleer factory: %w", err)
	}

	cont.lck.Lock()
	cont.factories = append(cont.factories, factory)
	cont.lck.Unlock()

	return nil
}

func provideContainer(ctx context.Context) (*container, error) {
	return appctx.Provide(ctx, containerCtxKey{}, func() (*container, error) {
		return &container{}, nil
	})
}
