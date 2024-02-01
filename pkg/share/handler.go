package share

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/guard"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/httpserver/crud"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/selm0/ladon"
)

type Shareable interface {
	db_repo.ModelBased
	GetResources() []string
	GetEntityType() string
}

type Metadata interface {
	GetOwnerId() uint
	GetActions() []string
}

type ModelBased interface {
	db_repo.ModelBased
	GetPolicyId() string
}

type BaseShareHandler interface {
	GetEntityModel() Shareable
	GetEntityRepository() db_repo.Repository
	GetGuard() guard.Guard
	GetModel() ModelBased
	GetRepository() db_repo.Repository
	TransformOutput(ctx context.Context, model db_repo.ModelBased, apiView string) (output interface{}, err error)
}

type EntityUpdateHandler interface {
	BaseShareHandler
	crud.BaseUpdateHandler
}

type EntityDeleteHandler interface {
	BaseShareHandler
}

type ShareCreateHandler interface {
	BaseShareHandler
	GetCreateInput() Metadata
	TransformCreate(ctx context.Context, input interface{}, entity Shareable, policy ladon.Policy, model db_repo.ModelBased) (err error)
}

func AddShareCreateHandler(logger log.Logger, d *httpserver.Definitions, version int, basePath string, handler ShareCreateHandler) {
	path := fmt.Sprintf("/v%d/%s/:id/share", version, basePath)
	d.POST(path, NewShareCreateHandler(logger, handler))
}
