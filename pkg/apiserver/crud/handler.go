package crud

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/apiserver"
	"github.com/applike/gosoline/pkg/db-repo"
	"github.com/jinzhu/inflection"
	"net/http"
)

const DefaultApiView = "api"

//go:generate mockery -name Repository
type Repository interface {
	Create(ctx context.Context, value db_repo.ModelBased) error
	Read(ctx context.Context, id *uint, out db_repo.ModelBased) error
	Update(ctx context.Context, value db_repo.ModelBased) error
	Delete(ctx context.Context, value db_repo.ModelBased) error
	Query(ctx context.Context, qb *db_repo.QueryBuilder, result interface{}) error
	Count(ctx context.Context, qb *db_repo.QueryBuilder, model db_repo.ModelBased) (int, error)
	GetMetadata() db_repo.Metadata
}

//go:generate mockery -name BaseHandler
type BaseHandler interface {
	GetRepository() Repository
	GetModel() db_repo.ModelBased
	TransformOutput(model db_repo.ModelBased, apiView string) (output interface{}, err error)
}

//go:generate mockery -name BaseCreateHandler
type BaseCreateHandler interface {
	GetCreateInput() interface{}
	TransformCreate(input interface{}, model db_repo.ModelBased) (err error)
}

//go:generate mockery -name CreateHandler
type CreateHandler interface {
	BaseHandler
	BaseCreateHandler
}

//go:generate mockery -name BaseUpdateHandler
type BaseUpdateHandler interface {
	GetUpdateInput() interface{}
	TransformUpdate(input interface{}, model db_repo.ModelBased) (err error)
}

//go:generate mockery -name UpdateHandler
type UpdateHandler interface {
	BaseHandler
	BaseUpdateHandler
}

//go:generate mockery -name BaseListHandler
type BaseListHandler interface {
	List(ctx context.Context, qb *db_repo.QueryBuilder, apiView string) (out interface{}, err error)
}

//go:generate mockery -name ListHandler
type ListHandler interface {
	BaseHandler
	BaseListHandler
}

//go:generate mockery -name DeleteHandler
type DeleteHandler interface {
	BaseHandler
	TransformDelete(model db_repo.ModelBased) (err error)
}

//go:generate mockery -name Handler
type Handler interface {
	BaseHandler
	BaseCreateHandler
	BaseUpdateHandler
	BaseListHandler
}

func AddCrudHandlers(d *apiserver.Definitions, version int, basePath string, handler Handler) {
	AddCreateHandler(d, version, basePath, handler)
	AddReadHandler(d, version, basePath, handler)
	AddUpdateHandler(d, version, basePath, handler)
	AddDeleteHandler(d, version, basePath, handler)
	AddListHandler(d, version, basePath, handler)
}

func AddCreateHandler(d *apiserver.Definitions, version int, basePath string, handler CreateHandler) {
	path, _ := getHandlerPaths(version, basePath)

	d.POST(path, NewCreateHandler(handler))
}

func AddReadHandler(d *apiserver.Definitions, version int, basePath string, handler BaseHandler) {
	_, idPath := getHandlerPaths(version, basePath)

	d.GET(idPath, NewReadHandler(handler))
}

func AddUpdateHandler(d *apiserver.Definitions, version int, basePath string, handler UpdateHandler) {
	_, idPath := getHandlerPaths(version, basePath)

	d.PUT(idPath, NewUpdateHandler(handler))
}

func AddDeleteHandler(d *apiserver.Definitions, version int, basePath string, handler BaseHandler) {
	_, idPath := getHandlerPaths(version, basePath)

	d.DELETE(idPath, NewDeleteHandler(handler))
}

func AddListHandler(d *apiserver.Definitions, version int, basePath string, handler ListHandler) {
	plural := inflection.Plural(basePath)
	path := fmt.Sprintf("/v%d/%s", version, plural)
	d.POST(path, NewListHandler(handler))
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
