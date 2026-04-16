package crud

import (
	"context"
	"errors"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/validation"
)

type patchInput = map[string]any

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
	uh := patchHandler{
		transformer: transformer,
		logger:      logger,
		settings:    settings,
	}

	return httpserver.CreateJsonHandler(uh), nil
}

func (ph patchHandler) GetInput() any {
	return new(patchInput)
}

func (ph patchHandler) Handle(reqCtx context.Context, request *httpserver.Request) (*httpserver.Response, error) {
	// replace context with a new one to prevent cancellations from client side
	// include a new timeout to ensure that requests will be cancelled
	ctx, cancel := exec.WithDelayedCancelContext(reqCtx, ph.settings.WriteTimeout)
	defer cancel()

	var ok bool
	var err error
	var id *uint
	var input *patchInput
	var updateInput any
	var before, patch, after []byte

	if id, ok = httpserver.GetUintFromRequest(request, "id"); !ok {
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

	// read model from repository
	if err = repo.Read(ctx, id, model); err != nil {
		return HandleErrorOnWrite(ctx, logger, err)
	}

	// transform model into the update input struct
	if updateInput, err = ph.transformer.TransformPatch(ctx, model); err != nil {
		return HandleErrorOnWrite(ctx, logger, fmt.Errorf("failed to transform patch: %w", err))
	}

	// marshal update input struct into bytes
	if before, err = json.Marshal(updateInput); err != nil {
		return HandleErrorOnWrite(ctx, logger, fmt.Errorf("failed to marshal model: %w", err))
	}

	// read patch as map[string]any
	if input, ok = request.Body.(*patchInput); !ok {
		return HandleErrorOnWrite(ctx, logger, fmt.Errorf("invalid request body type: %T", request.Body))
	}

	// marshal patch into bytes
	if patch, err = json.Marshal(input); err != nil {
		return HandleErrorOnWrite(ctx, logger, fmt.Errorf("failed to marshal input: %w", err))
	}

	// apply patch to update input bytes
	if after, err = jsonpatch.MergePatch(before, patch); err != nil {
		return HandleErrorOnWrite(ctx, logger, fmt.Errorf("failed to merge patch: %w", err))
	}

	// unmarshal patched bytes into update input struct
	if err = json.Unmarshal(after, updateInput); err != nil {
		return HandleErrorOnWrite(ctx, logger, fmt.Errorf("failed to unmarshal patched model: %w", err))
	}

	// apply update input struct onto model
	if err = ph.transformer.TransformUpdate(ctx, updateInput, model); err != nil {
		return HandleErrorOnWrite(ctx, logger, fmt.Errorf("failed to transform update: %w", err))
	}

	// write model back to repository
	if err = repo.Update(ctx, model); err != nil {
		return HandleErrorOnWrite(ctx, logger, err)
	}

	reload := ph.transformer.GetModel()
	if err = repo.Read(ctx, model.GetId(), reload); err != nil {
		return HandleErrorOnWrite(ctx, logger, err)
	}

	apiView := GetApiViewFromHeader(request.Header)
	out, err := ph.transformer.TransformOutput(ctx, reload, apiView)
	if err != nil {
		return HandleErrorOnWrite(ctx, logger, err)
	}

	return httpserver.NewJsonResponse(out), nil
}
