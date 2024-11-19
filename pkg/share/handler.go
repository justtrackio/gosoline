package share

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/guard"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/httpserver/crud"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/selm0/ladon"
)

type Shareable[K mdl.PossibleIdentifier] interface {
	db_repo.ModelBased[K]
	GetResources() []string
	GetEntityType() string
}

type Metadata interface {
	GetOwnerId() uint
	GetActions() []string
}

type ModelBased[K mdl.PossibleIdentifier] interface {
	db_repo.ModelBased[K]
	GetPolicyId() string
}

type BaseShareHandler[K mdl.PossibleIdentifier, M Shareable[K]] interface {
	GetEntityRepository() db_repo.Repository[K, M]
	GetGuard() guard.Guard
	GetRepository() db_repo.Repository[K, M]
	TransformOutput(ctx context.Context, model M, apiView string) (output any, err error)
}

type EntityUpdateHandler[I any, O any, K mdl.PossibleIdentifier, M Shareable[K]] interface {
	BaseShareHandler[K, M]
	crud.BaseUpdateHandler[I, K, M]
}

type EntityDeleteHandler[K mdl.PossibleIdentifier, M Shareable[K]] interface {
	BaseShareHandler[K, M]
}

type ShareCreateHandler[I Metadata, K mdl.PossibleIdentifier, M Shareable[K]] interface {
	BaseShareHandler[K, M]
	TransformCreate(ctx context.Context, input I, entity Shareable[K], policy ladon.Policy) (model M, err error)
}

func AddShareCreateHandler[I Metadata, K mdl.PossibleIdentifier, M Shareable[K]](logger log.Logger, d *httpserver.Definitions, version int, basePath string, handler ShareCreateHandler[I, K, M]) {
	path := fmt.Sprintf("/v%d/%s/:id/share", version, basePath)
	d.POST(path, NewShareCreateHandler[I, K, M](logger, handler))
}
