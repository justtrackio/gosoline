package crud

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jinzhu/inflection"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	SettingsConfigKey = "crud"
	DefaultApiView    = "api"
)

// Settings structure for all CRUDL handler.
type Settings struct {
	// Applies to create, update and delete handlers.
	// Write timeout is the maximum duration before canceling any write operation.
	WriteTimeout time.Duration `cfg:"write_timeout" default:"10min" validate:"min=1000000000"`
}

//go:generate mockery --name Repository
type Repository interface {
	Create(ctx context.Context, value db_repo.ModelBased) error
	Read(ctx context.Context, id *uint, out db_repo.ModelBased) error
	Update(ctx context.Context, value db_repo.ModelBased) error
	Delete(ctx context.Context, value db_repo.ModelBased) error
	Query(ctx context.Context, qb *db_repo.QueryBuilder, result any) error
	Count(ctx context.Context, qb *db_repo.QueryBuilder, model db_repo.ModelBased) (int, error)
	GetMetadata() db_repo.Metadata
}

//go:generate mockery --name BaseHandler
type BaseHandler interface {
	GetRepository() Repository
	GetModel() db_repo.ModelBased
	TransformOutput(ctx context.Context, model db_repo.ModelBased, apiView string) (output any, err error)
}

//go:generate mockery --name BaseCreateHandler
type BaseCreateHandler interface {
	GetCreateInput() any
	TransformCreate(ctx context.Context, input any, model db_repo.ModelBased) (err error)
}

//go:generate mockery --name CreateHandler
type CreateHandler interface {
	BaseHandler
	BaseCreateHandler
}

//go:generate mockery --name BaseUpdateHandler
type BaseUpdateHandler interface {
	GetUpdateInput() any
	TransformUpdate(ctx context.Context, input any, model db_repo.ModelBased) (err error)
}

//go:generate mockery --name UpdateHandler
type UpdateHandler interface {
	BaseHandler
	BaseUpdateHandler
}

//go:generate mockery --name BaseListHandler
type BaseListHandler interface {
	List(ctx context.Context, qb *db_repo.QueryBuilder, apiView string) (out any, err error)
}

//go:generate mockery --name ListHandler
type ListHandler interface {
	BaseHandler
	BaseListHandler
}

//go:generate mockery --name Handler
type Handler interface {
	BaseHandler
	BaseCreateHandler
	BaseUpdateHandler
	BaseListHandler
}

func AddCrudHandlers(config cfg.Config, logger log.Logger, d *httpserver.Definitions, version int, basePath string, handler Handler) {
	AddCreateHandler(config, logger, d, version, basePath, handler)
	AddReadHandler(config, logger, d, version, basePath, handler)
	AddUpdateHandler(config, logger, d, version, basePath, handler)
	AddDeleteHandler(config, logger, d, version, basePath, handler)
	AddListHandler(config, logger, d, version, basePath, handler)
}

func AddCreateHandler(config cfg.Config, logger log.Logger, d *httpserver.Definitions, version int, basePath string, handler CreateHandler) {
	path, _ := getHandlerPaths(version, basePath)

	d.POST(path, NewCreateHandler(config, logger, handler))
}

func AddReadHandler(config cfg.Config, logger log.Logger, d *httpserver.Definitions, version int, basePath string, handler BaseHandler) {
	_, idPath := getHandlerPaths(version, basePath)

	d.GET(idPath, NewReadHandler(config, logger, handler))
}

func AddUpdateHandler(config cfg.Config, logger log.Logger, d *httpserver.Definitions, version int, basePath string, handler UpdateHandler) {
	_, idPath := getHandlerPaths(version, basePath)

	d.PUT(idPath, NewUpdateHandler(config, logger, handler))
}

func AddDeleteHandler(config cfg.Config, logger log.Logger, d *httpserver.Definitions, version int, basePath string, handler BaseHandler) {
	_, idPath := getHandlerPaths(version, basePath)

	d.DELETE(idPath, NewDeleteHandler(config, logger, handler))
}

func AddListHandler(config cfg.Config, logger log.Logger, d *httpserver.Definitions, version int, basePath string, handler ListHandler) {
	plural := inflection.Plural(basePath)
	path := fmt.Sprintf("/v%d/%s", version, plural)
	d.POST(path, NewListHandler(config, logger, handler))
}

func getHandlerPaths(version int, basePath string) (path string, idPath string) {
	path = fmt.Sprintf("/v%d/%s", version, basePath)
	idPath = fmt.Sprintf("%s/:id", path)

	return
}

func GetApiViewFromHeader(reqHeaders http.Header) string {
	if apiView := reqHeaders.Get(httpserver.ApiViewKey); apiView != "" {
		return apiView
	}

	return DefaultApiView
}
