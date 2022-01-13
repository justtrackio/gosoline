package crud

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/apiserver"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/validation"
)

type updateHandler struct {
	logger      log.Logger
	transformer UpdateHandler
}

func NewUpdateHandler(logger log.Logger, transformer UpdateHandler) gin.HandlerFunc {
	uh := updateHandler{
		transformer: transformer,
		logger:      logger,
	}

	return apiserver.CreateJsonHandler(uh)
}

func (uh updateHandler) GetInput() interface{} {
	return uh.transformer.GetUpdateInput()
}

func (uh updateHandler) Handle(ctx context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	id, valid := apiserver.GetUintFromRequest(request, "id")

	if !valid {
		return nil, errors.New("no valid id provided")
	}

	repo := uh.transformer.GetRepository()
	model := uh.transformer.GetModel()
	err := repo.Read(ctx, id, model)

	var notFound db_repo.RecordNotFoundError
	if errors.As(err, &notFound) {
		uh.logger.WithContext(ctx).Warn("failed to update model: %s", err)

		return apiserver.NewStatusResponse(http.StatusNotFound), nil
	}

	if err != nil {
		return nil, err
	}

	err = uh.transformer.TransformUpdate(ctx, request.Body, model)

	if modelNotChanged(err) {
		return apiserver.NewStatusResponse(http.StatusNotModified), nil
	}

	if err != nil {
		return nil, err
	}

	err = repo.Update(ctx, model)

	if db.IsDuplicateEntryError(err) {
		return apiserver.NewStatusResponse(http.StatusConflict), nil
	}

	if errors.Is(err, &validation.Error{}) {
		return apiserver.GetErrorHandler()(http.StatusBadRequest, err), nil
	}

	if err != nil {
		return nil, err
	}

	reload := uh.transformer.GetModel()
	err = repo.Read(ctx, model.GetId(), reload)

	if err != nil {
		return nil, err
	}

	apiView := GetApiViewFromHeader(request.Header)
	out, err := uh.transformer.TransformOutput(ctx, reload, apiView)
	if err != nil {
		return nil, err
	}

	return apiserver.NewJsonResponse(out), nil
}

func modelNotChanged(err error) bool {
	return errors.Is(err, ErrModelNotChanged)
}
