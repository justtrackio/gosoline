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

type updateHandler[I any, O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]] struct {
	logger      log.Logger
	transformer UpdateHandler[I, O, K, M]
	settings    Settings
}

func NewUpdateHandler[I any, O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]](config cfg.Config, logger log.Logger, transformer UpdateHandler[I, O, K, M]) (gin.HandlerFunc, error) {
	settings := Settings{}
	if err := config.UnmarshalKey(SettingsConfigKey, &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal update handler settings: %w", err)
	}

	uh := updateHandler[I, O, K, M]{
		logger:      logger,
		transformer: transformer,
		settings:    settings,
	}

	return httpserver.CreateJsonHandler(uh), nil
}

func (uh updateHandler[I, O, K, M]) GetInput() any {
	var input I

	return &input
}

func (uh updateHandler[I, O, K, M]) Handle(ctx context.Context, request *httpserver.Request) (*httpserver.Response, error) {
	// replace context with a new one to prevent cancellations from client side
	// include a new timeout to ensure that requests will be cancelled
	ctx, cancel := exec.WithDelayedCancelContext(ctx, uh.settings.WriteTimeout)
	defer cancel()

	id, valid := httpserver.GetIdentifierFromRequest[K](request, "id")

	if !valid {
		return HandleErrorOnWrite(ctx, uh.logger, &validation.Error{
			Errors: []error{
				errors.New("no valid id provided"),
			},
		})
	}

	logger := uh.logger.WithFields(log.Fields{
		"entity_id": id,
	})

	repo := uh.transformer.GetRepository()
	model, err := repo.Read(ctx, *id)

	if err != nil {
		return HandleErrorOnWrite(ctx, logger, err)
	}

	model, err = uh.transformer.TransformUpdate(ctx, request.Body.(*I), model)

	if err != nil {
		return HandleErrorOnWrite(ctx, logger, err)
	}

	err = repo.Update(ctx, model)
	if err != nil {
		return HandleErrorOnWrite(ctx, logger, err)
	}

	reload, err := repo.Read(ctx, *model.GetId())
	if err != nil {
		return HandleErrorOnWrite(ctx, logger, err)
	}

	apiView := GetApiViewFromHeader(request.Header)
	out, err := uh.transformer.TransformOutput(ctx, reload, apiView)
	if err != nil {
		return HandleErrorOnWrite(ctx, logger, err)
	}

	return httpserver.NewJsonResponse(out), nil
}
