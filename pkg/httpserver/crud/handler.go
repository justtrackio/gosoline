package crud

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jinzhu/inflection"
	"github.com/justtrackio/gosoline/pkg/cfg"
	dbRepo "github.com/justtrackio/gosoline/pkg/db-repo"
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
	WriteTimeout time.Duration `cfg:"write_timeout" default:"10m" validate:"min=1000000000"`
}

//go:generate go run github.com/vektra/mockery/v2 --name Repository
type Repository interface {
	Create(ctx context.Context, value dbRepo.ModelBased) error
	Read(ctx context.Context, id *uint, out dbRepo.ModelBased) error
	Update(ctx context.Context, value dbRepo.ModelBased) error
	Delete(ctx context.Context, value dbRepo.ModelBased) error
	Query(ctx context.Context, qb *dbRepo.QueryBuilder, result any) error
	Count(ctx context.Context, qb *dbRepo.QueryBuilder, model dbRepo.ModelBased) (int, error)
	GetMetadata() dbRepo.Metadata
}

//go:generate go run github.com/vektra/mockery/v2 --name BaseHandler
type BaseHandler interface {
	GetRepository() Repository
	GetModel() dbRepo.ModelBased
	TransformOutput(ctx context.Context, model dbRepo.ModelBased, apiView string) (output any, err error)
}

//go:generate go run github.com/vektra/mockery/v2 --name BaseCreateHandler
type BaseCreateHandler interface {
	GetCreateInput() any
	TransformCreate(ctx context.Context, input any, model dbRepo.ModelBased) (err error)
}

//go:generate go run github.com/vektra/mockery/v2 --name CreateHandler
type CreateHandler interface {
	BaseHandler
	BaseCreateHandler
}

//go:generate go run github.com/vektra/mockery/v2 --name BaseUpdateHandler
type BaseUpdateHandler interface {
	GetUpdateInput() any
	TransformUpdate(ctx context.Context, input any, model dbRepo.ModelBased) (err error)
}

//go:generate go run github.com/vektra/mockery/v2 --name UpdateHandler
type UpdateHandler interface {
	BaseHandler
	BaseUpdateHandler
}

//go:generate go run github.com/vektra/mockery/v2 --name BaseListHandler
type BaseListHandler interface {
	List(ctx context.Context, qb *dbRepo.QueryBuilder, apiView string) (out any, err error)
}

//go:generate go run github.com/vektra/mockery/v2 --name ListHandler
type ListHandler interface {
	BaseHandler
	BaseListHandler
}

//go:generate go run github.com/vektra/mockery/v2 --name Handler
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
