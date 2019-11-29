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

//go:generate mockery -name Handler
type Handler interface {
	GetRepository() Repository
	GetModel() db_repo.ModelBased
	GetCreateInput() interface{}
	GetUpdateInput() interface{}
	TransformCreate(input interface{}, model db_repo.ModelBased) (err error)
	TransformUpdate(input interface{}, model db_repo.ModelBased) (err error)
	TransformOutput(model db_repo.ModelBased, apiView string) (output interface{}, err error)
	List(ctx context.Context, qb *db_repo.QueryBuilder, apiView string) (out interface{}, err error)
}

func AddCrudHandlers(d *apiserver.Definitions, version int, basePath string, handler Handler) {
	path := fmt.Sprintf("/v%d/%s", version, basePath)
	idPath := fmt.Sprintf("%s/:id", path)

	d.POST(path, NewCreateHandler(handler))
	d.GET(idPath, NewReadHandler(handler))
	d.PUT(idPath, NewUpdateHandler(handler))
	d.DELETE(idPath, NewDeleteHandler(handler))

	plural := inflection.Plural(basePath)
	path = fmt.Sprintf("/v%d/%s", version, plural)
	d.POST(path, NewListHandler(handler))
}

func GetApiViewFromHeader(reqHeaders http.Header) string {
	if apiView := reqHeaders.Get(apiserver.ApiViewKey); apiView != "" {
		return apiView
	}

	return DefaultApiView
}
