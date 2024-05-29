package transactions

import (
	"context"
	"fmt"

	"github.com/beeemT/go-atomic"
	"github.com/beeemT/go-atomic/generic"
	"github.com/beeemT/go-atomic/generic/gorm"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	db_repo "github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/log"
)

var _ atomic.Transacter[struct{}] = Transacter[struct{}]{}

type (
	Transacter[Resources any] struct {
		generic.Transacter[db_repo.Remote, Resources]
	}

	transacterKey string
)

func NewTransacter[Resources any](
	ctx context.Context,
	config cfg.Config,
	logger log.Logger,
	resourcesFactory func(
		ctx context.Context,
		transacter *generic.Transacter[db_repo.Remote, Resources],
		tx db_repo.Remote,
	) (Resources, error),
) (*Transacter[Resources], error) {
	orm, err := db_repo.NewConfiguredOrm(config, logger)
	if err != nil {
		return nil, fmt.Errorf("creating orm: %w", err)
	}

	transacter := generic.NewTransacter(
		gorm.NewExecuter(&db_repo.Gorm{DB: orm}),
		resourcesFactory,
	)

	return &Transacter[Resources]{
		Transacter: transacter,
	}, nil
}

func (transacter Transacter[Resources]) Transact(ctx context.Context, run func(context.Context, Resources) error) (err error) {
	return transacter.Transacter.Transact(ctx, run)
}

func ProvideTransacter[Resources any](
	ctx context.Context,
	config cfg.Config,
	logger log.Logger,
	resourcesFactory func(
		ctx context.Context,
		transacter *generic.Transacter[db_repo.Remote, Resources],
		tx db_repo.Remote,
	) (Resources, error),
) (atomic.Transacter[Resources], error) {
	return appctx.Provide(ctx, transacterKey("transacter"), func() (atomic.Transacter[Resources], error) {
		return NewTransacter(ctx, config, logger, resourcesFactory)
	})
}
