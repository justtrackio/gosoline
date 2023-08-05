package crud

import (
	"context"
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/cfg"
	dbRepo "github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/validation"
)

type deleteHandler[O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]] struct {
	logger      log.Logger
	transformer BaseHandler[O, K, M]
	settings    Settings
}

func NewDeleteHandler[O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]](config cfg.Config, logger log.Logger, transformer BaseHandler[O, K, M]) (gin.HandlerFunc, error) {
	settings := Settings{}
	if err := config.UnmarshalKey(SettingsConfigKey, &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal delete handler settings: %w", err)
	}

	dh := deleteHandler[O, K, M]{
		logger:      logger,
		transformer: transformer,
		settings:    settings,
	}

	return httpserver.CreateHandler(dh), nil
}

func (dh deleteHandler[O, K, M]) Handle(ctx context.Context, request *httpserver.Request) (*httpserver.Response, error) {
	// replace context with a new one to prevent cancellations from client side
	// include a new timeout to ensure that requests will be cancelled
	ctx, cancel := exec.WithDelayedCancelContext(ctx, dh.settings.WriteTimeout)
	defer cancel()

	id, valid := httpserver.GetIdentifierFromRequest[K](request, "id")
	if !valid {
		return HandleErrorOnWrite(ctx, dh.logger, &validation.Error{
			Errors: []error{
				errors.New("no valid id provided"),
			},
		})
	}

	repo := dh.transformer.GetRepository()

	model, err := repo.Read(ctx, *id)
	if err != nil {
		return HandleErrorOnWrite(ctx, dh.logger, err)
	}

	err = repo.Delete(ctx, model)
	if err != nil {
		return HandleErrorOnWrite(ctx, dh.logger, err)
	}

	apiView := GetApiViewFromHeader(request.Header)
	out, err := dh.transformer.TransformOutput(ctx, model, apiView)
	if err != nil {
		return HandleErrorOnWrite(ctx, dh.logger, err)
	}

	return httpserver.NewJsonResponse(out), nil
}
