package crud

import (
	"context"
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/validation"
)

type patchHandler struct {
	logger      log.Logger
	transformer PatchHandler
	settings    Settings
}

func NewPatchHandler(config cfg.Config, logger log.Logger, transformer PatchHandler) (gin.HandlerFunc, error) {
	settings := Settings{}
	if err := config.UnmarshalKey(SettingsConfigKey, &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal patch handler settings: %w", err)
	}

	ph := patchHandler{
		transformer: transformer,
		logger:      logger,
		settings:    settings,
	}

	return httpserver.CreateJsonHandler(ph), nil
}

func (ph patchHandler) GetInput() any {
	return ph.transformer.GetPatchInput()
}

func (ph patchHandler) Handle(reqCtx context.Context, request *httpserver.Request) (*httpserver.Response, error) {
	// replace context with a new one to prevent cancellations from client side
	// include a new timeout to ensure that requests will be cancelled
	ctx, cancel := exec.WithDelayedCancelContext(reqCtx, ph.settings.WriteTimeout)
	defer cancel()

	id, valid := httpserver.GetUintFromRequest(request, "id")

	if !valid {
		return HandleErrorOnWrite(ctx, ph.logger, &validation.Error{
			Errors: []error{
				errors.New("no valid id provided"),
			},
		})
	}

	logger := ph.logger.WithFields(log.Fields{
		"entity_id": id,
	})

	repo := ph.transformer.GetRepository()
	model := ph.transformer.GetModel()

	err := repo.Read(ctx, id, model)
	if err != nil {
		return HandleErrorOnWrite(ctx, logger, err)
	}

	err = ph.transformer.TransformPatch(ctx, request.Body, model)
	if err != nil {
		return HandleErrorOnWrite(ctx, logger, err)
	}

	// model and request.Body to map[string]any and merge?

	err = repo.Update(ctx, model)
	if err != nil {
		return HandleErrorOnWrite(ctx, logger, err)
	}

	reload := ph.transformer.GetModel()
	err = repo.Read(ctx, model.GetId(), reload)
	if err != nil {
		return HandleErrorOnWrite(ctx, logger, err)
	}

	apiView := GetApiViewFromHeader(request.Header)
	out, err := ph.transformer.TransformOutput(ctx, reload, apiView)
	if err != nil {
		return HandleErrorOnWrite(ctx, logger, err)
	}

	return httpserver.NewJsonResponse(out), nil
}
