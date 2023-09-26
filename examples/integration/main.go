//go:build integration && fixtures

package integration

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type item struct {
	Id        uint      `json:"id" ddb:"key=hash"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (i *item) GetId() *uint {
	return &i.Id
}

func (i *item) SetUpdatedAt(updatedAt *time.Time) {
	i.UpdatedAt = mdl.EmptyIfNil(updatedAt)
}

func (i *item) SetCreatedAt(createdAt *time.Time) {
	i.UpdatedAt = mdl.EmptyIfNil(createdAt)
}

var ddbSettings = &ddb.Settings{
	Main: ddb.MainSettings{
		Model:              &item{},
		ReadCapacityUnits:  1,
		WriteCapacityUnits: 1,
	},
}

var repoSettings = db_repo.Settings{
	AppId: cfg.AppId{},
	Metadata: db_repo.Metadata{
		TableName: "items",
	},
}

type app struct {
	kernel.EssentialModule
	kernel.ServiceStage
	ddbRepository ddb.Repository
	dbRepository  db_repo.Repository
}

func newAppModule(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	ddbRepository, err := ddb.NewRepository(ctx, config, logger, ddbSettings)
	if err != nil {
		return nil, fmt.Errorf("unable to create dynamodb repository: %w", err)
	}

	dbRepository, err := db_repo.New(ctx, config, logger, repoSettings)
	if err != nil {
		return nil, fmt.Errorf("unable to create mysql client: %w", err)
	}

	return &app{
		ddbRepository: ddbRepository,
		dbRepository:  dbRepository,
	}, nil
}

func (a app) Run(ctx context.Context) error {
	item := &item{
		Id:        1,
		CreatedAt: clock.Provider.Now(),
		UpdatedAt: clock.Provider.Now(),
	}

	qb := a.ddbRepository.
		PutItemBuilder()

	_, err := a.ddbRepository.PutItem(ctx, qb, item)
	if err != nil {
		return fmt.Errorf("cannot put item to dynamodb: %w", err)
	}

	err = a.dbRepository.Create(ctx, item)
	if err != nil {
		return fmt.Errorf("cannot create item in db: %w", err)
	}

	return nil
}
