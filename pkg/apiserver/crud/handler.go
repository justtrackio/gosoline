package crud

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jinzhu/inflection"
	"github.com/justtrackio/gosoline/pkg/apiserver"
	dbRepo "github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

const DefaultApiView = "api"

//go:generate mockery --name BaseHandler
type BaseHandler[O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]] interface {
	GetRepository() dbRepo.Repository[K, M]
	TransformOutput(ctx context.Context, model M, apiView string) (output O, err error)
}

//go:generate mockery --name BaseCreateHandler
type BaseCreateHandler[I any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]] interface {
	TransformCreate(ctx context.Context, input *I) (M, error)
}

//go:generate mockery --name CreateHandler
type CreateHandler[I any, O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]] interface {
	BaseHandler[O, K, M]
	BaseCreateHandler[I, K, M]
}

//go:generate mockery --name BaseUpdateHandler
type BaseUpdateHandler[I any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]] interface {
	TransformUpdate(ctx context.Context, input *I, model M) (M, error)
}

//go:generate mockery --name UpdateHandler
type UpdateHandler[I any, O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]] interface {
	BaseHandler[O, K, M]
	BaseUpdateHandler[I, K, M]
}

//go:generate mockery --name BaseListHandler
type BaseListHandler[O any] interface {
	List(ctx context.Context, qb *dbRepo.QueryBuilder, apiView string) (out []O, err error)
}

//go:generate mockery --name ListHandler
type ListHandler[O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]] interface {
	BaseHandler[O, K, M]
	BaseListHandler[O]
}

//go:generate mockery --name Handler
type Handler[CI any, UI any, O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]] interface {
	BaseHandler[O, K, M]
	BaseCreateHandler[CI, K, M]
	BaseUpdateHandler[UI, K, M]
	BaseListHandler[O]
}

func AddCrudHandlers[CI any, UI any, O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]](logger log.Logger, d *apiserver.Definitions, version int, basePath string, handler Handler[CI, UI, O, K, M]) {
	AddCreateHandler[CI, O, K, M](logger, d, version, basePath, handler)
	AddReadHandler[O, K, M](logger, d, version, basePath, handler)
	AddUpdateHandler[UI, O, K, M](logger, d, version, basePath, handler)
	AddDeleteHandler[O, K, M](logger, d, version, basePath, handler)
	AddListHandler[O, K, M](logger, d, version, basePath, handler)
}

func AddCreateHandler[I any, O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]](logger log.Logger, d *apiserver.Definitions, version int, basePath string, handler CreateHandler[I, O, K, M]) {
	path, _ := getHandlerPaths(version, basePath)

	d.POST(path, NewCreateHandler(logger, handler))
}

func AddReadHandler[O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]](logger log.Logger, d *apiserver.Definitions, version int, basePath string, handler BaseHandler[O, K, M]) {
	_, idPath := getHandlerPaths(version, basePath)

	d.GET(idPath, NewReadHandler(logger, handler))
}

func AddUpdateHandler[I any, O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]](logger log.Logger, d *apiserver.Definitions, version int, basePath string, handler UpdateHandler[I, O, K, M]) {
	_, idPath := getHandlerPaths(version, basePath)

	d.PUT(idPath, NewUpdateHandler(logger, handler))
}

func AddDeleteHandler[O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]](logger log.Logger, d *apiserver.Definitions, version int, basePath string, handler BaseHandler[O, K, M]) {
	_, idPath := getHandlerPaths(version, basePath)

	d.DELETE(idPath, NewDeleteHandler(logger, handler))
}

func AddListHandler[O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]](logger log.Logger, d *apiserver.Definitions, version int, basePath string, handler ListHandler[O, K, M]) {
	plural := inflection.Plural(basePath)
	path := fmt.Sprintf("/v%d/%s", version, plural)
	d.POST(path, NewListHandler(logger, handler))
}

func getHandlerPaths(version int, basePath string) (path string, idPath string) {
	path = fmt.Sprintf("/v%d/%s", version, basePath)
	idPath = fmt.Sprintf("%s/:id", path)

	return
}

func GetApiViewFromHeader(reqHeaders http.Header) string {
	if apiView := reqHeaders.Get(apiserver.ApiViewKey); apiView != "" {
		return apiView
	}

	return DefaultApiView
}
