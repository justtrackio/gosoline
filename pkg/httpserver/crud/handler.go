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
	"github.com/justtrackio/gosoline/pkg/mdl"
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

//go:generate go run github.com/vektra/mockery/v2 --name BaseHandler
type BaseHandler[O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]] interface {
	GetRepository() dbRepo.Repository[K, M]
	TransformOutput(ctx context.Context, model M, apiView string) (output O, err error)
}

//go:generate go run github.com/vektra/mockery/v2 --name BaseCreateHandler
type BaseCreateHandler[I any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]] interface {
	TransformCreate(ctx context.Context, input *I) (M, error)
}

//go:generate go run github.com/vektra/mockery/v2 --name CreateHandler
type CreateHandler[I any, O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]] interface {
	BaseHandler[O, K, M]
	BaseCreateHandler[I, K, M]
}

//go:generate go run github.com/vektra/mockery/v2 --name BaseUpdateHandler
type BaseUpdateHandler[I any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]] interface {
	TransformUpdate(ctx context.Context, input *I, model M) (M, error)
}

//go:generate go run github.com/vektra/mockery/v2 --name UpdateHandler
type UpdateHandler[I any, O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]] interface {
	BaseHandler[O, K, M]
	BaseUpdateHandler[I, K, M]
}

//go:generate go run github.com/vektra/mockery/v2 --name BaseListHandler
type BaseListHandler[O any] interface {
	List(ctx context.Context, qb *dbRepo.QueryBuilder, apiView string) (out []O, err error)
}

//go:generate go run github.com/vektra/mockery/v2 --name ListHandler
type ListHandler[O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]] interface {
	BaseHandler[O, K, M]
	BaseListHandler[O]
}

//go:generate go run github.com/vektra/mockery/v2 --name Handler
type Handler[CI any, UI any, O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]] interface {
	BaseHandler[O, K, M]
	BaseCreateHandler[CI, K, M]
	BaseUpdateHandler[UI, K, M]
	BaseListHandler[O]
}

func AddCrudHandlers[CI any, UI any, O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]](config cfg.Config, logger log.Logger, d *httpserver.Definitions, version int, basePath string, handler Handler[CI, UI, O, K, M]) error {
	if err := AddCreateHandler[CI, O, K, M](config, logger, d, version, basePath, handler); err != nil {
		return fmt.Errorf("failed to add create handler: %w", err)
	}

	AddReadHandler[O, K, M](logger, d, version, basePath, handler)

	if err := AddUpdateHandler[UI, O, K, M](config, logger, d, version, basePath, handler); err != nil {
		return fmt.Errorf("failed to add update handler: %w", err)
	}

	if err := AddDeleteHandler[O, K, M](config, logger, d, version, basePath, handler); err != nil {
		return fmt.Errorf("failed to add delete handler: %w", err)
	}

	AddListHandler[O, K, M](logger, d, version, basePath, handler)

	return nil
}

func AddCreateHandler[I any, O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]](config cfg.Config, logger log.Logger, d *httpserver.Definitions, version int, basePath string, handler CreateHandler[I, O, K, M]) error {
	path, _ := getHandlerPaths(version, basePath)

	createHandler, err := NewCreateHandler(config, logger, handler)
	if err != nil {
		return fmt.Errorf("failed to create handler: %w", err)
	}

	d.POST(path, createHandler)

	return nil
}

func AddReadHandler[O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]](logger log.Logger, d *httpserver.Definitions, version int, basePath string, handler BaseHandler[O, K, M]) {
	_, idPath := getHandlerPaths(version, basePath)

	d.GET(idPath, NewReadHandler(logger, handler))
}

func AddUpdateHandler[I any, O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]](config cfg.Config, logger log.Logger, d *httpserver.Definitions, version int, basePath string, handler UpdateHandler[I, O, K, M]) error {
	_, idPath := getHandlerPaths(version, basePath)

	updateHandler, err := NewUpdateHandler(config, logger, handler)
	if err != nil {
		return fmt.Errorf("failed to create update handler: %w", err)
	}

	d.PUT(idPath, updateHandler)

	return nil
}

func AddDeleteHandler[O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]](config cfg.Config, logger log.Logger, d *httpserver.Definitions, version int, basePath string, handler BaseHandler[O, K, M]) error {
	_, idPath := getHandlerPaths(version, basePath)

	deleteHandler, err := NewDeleteHandler(config, logger, handler)
	if err != nil {
		return fmt.Errorf("failed to create delete handler: %w", err)
	}

	d.DELETE(idPath, deleteHandler)

	return nil
}

func AddListHandler[O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]](logger log.Logger, d *httpserver.Definitions, version int, basePath string, handler ListHandler[O, K, M]) {
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
	if apiView := reqHeaders.Get(httpserver.ApiViewKey); apiView != "" {
		return apiView
	}

	return DefaultApiView
}
