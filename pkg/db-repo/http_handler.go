package db_repo

import (
	"context"
	"fmt"
	"net/http"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type RepoHandler[T any] struct {
	repo RepositoryReadOnly
}

func ProvideModelHandler[T any](ctx context.Context, config cfg.Config, logger log.Logger, metadata *Metadata) (*RepoHandler[T], error) {
	return appctx.Provide(ctx, RepoHandler[T]{}, func() (*RepoHandler[T], error) {
		return NewModelHandler[T](ctx, config, logger, metadata)
	})
}

func NewModelHandler[T any](_ context.Context, config cfg.Config, logger log.Logger, metadata *Metadata) (*RepoHandler[T], error) {
	var err error
	handler := &RepoHandler[T]{}

	settings := Settings{
		Metadata: *metadata,
	}

	if handler.repo, err = New(config, logger, settings); err != nil {
		return nil, fmt.Errorf("can not create db_repo.Repository for model %s: %w", metadata.ModelId, err)
	}

	return handler, nil
}

func (h *RepoHandler[T]) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	var err error

	result := make([]T, 0)
	qb := NewQueryBuilder()

	if err = h.repo.Query(request.Context(), qb, &result); err != nil {
		panic(err)
	}

	fmt.Println("got it")
}
