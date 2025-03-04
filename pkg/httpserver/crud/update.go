package crud

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/validation"
)

type updateHandler struct {
	logger      log.Logger
	transformer UpdateHandler
	settings    Settings
}

func NewUpdateHandler(config cfg.Config, logger log.Logger, transformer UpdateHandler) gin.HandlerFunc {
	settings := Settings{}
	config.UnmarshalKey(SettingsConfigKey, &settings)

	uh := updateHandler{
		transformer: transformer,
		logger:      logger,
		settings:    settings,
	}

	return httpserver.CreateJsonHandler(uh)
}

func (uh updateHandler) GetInput() any {
	return uh.transformer.GetUpdateInput()
}

func (uh updateHandler) Handle(reqCtx context.Context, request *httpserver.Request) (*httpserver.Response, error) {
	// replace context with a new one to prevent cancellations from client side
	// include a new timeout to ensure that requests will be cancelled
	ctx, cancel := exec.WithDelayedCancelContext(reqCtx, uh.settings.WriteTimeout)
	defer cancel()

	logger := uh.logger.WithContext(ctx)

	id, valid := httpserver.GetUintFromRequest(request, "id")

	if !valid {
		return HandleErrorOnWrite(ctx, logger, &validation.Error{
			Errors: []error{
				errors.New("no valid id provided"),
			},
		})
	}

	logger = logger.WithFields(log.Fields{
		"entity_id": id,
	})

	repo := uh.transformer.GetRepository()
	model := uh.transformer.GetModel()

	err := repo.Read(ctx, id, model)
	if err != nil {
		return HandleErrorOnWrite(ctx, logger, err)
	}

	err = uh.transformer.TransformUpdate(ctx, request.Body, model)
	if err != nil {
		return HandleErrorOnWrite(ctx, logger, err)
	}

	err = repo.Update(ctx, model)
	if err != nil {
		return HandleErrorOnWrite(ctx, logger, err)
	}

	reload := uh.transformer.GetModel()
	err = repo.Read(ctx, model.GetId(), reload)
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
