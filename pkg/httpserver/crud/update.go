package crud

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/db"
	dbRepo "github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/validation"
)

type updateHandler[I any, O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]] struct {
	logger      log.Logger
	transformer UpdateHandler[I, O, K, M]
}

func NewUpdateHandler[I any, O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]](logger log.Logger, transformer UpdateHandler[I, O, K, M]) gin.HandlerFunc {
	uh := updateHandler[I, O, K, M]{
		transformer: transformer,
		logger:      logger,
	}

	return httpserver.CreateJsonHandler(uh)
}

func (uh updateHandler[I, O, K, M]) GetInput() any {
	var input I

	return &input
}

func (uh updateHandler[I, O, K, M]) Handle(ctx context.Context, request *httpserver.Request) (*httpserver.Response, error) {
	id, valid := httpserver.GetIdentifierFromRequest[K](request, "id")

	if !valid {
		return nil, errors.New("no valid id provided")
	}

	repo := uh.transformer.GetRepository()
	model, err := repo.Read(ctx, *id)

	var notFound dbRepo.RecordNotFoundError
	if errors.As(err, &notFound) {
		uh.logger.WithContext(ctx).Warn("failed to update model: %s", err.Error())

		return httpserver.NewStatusResponse(http.StatusNotFound), nil
	}

	if err != nil {
		return nil, err
	}

	model, err = uh.transformer.TransformUpdate(ctx, request.Body.(*I), model)

	if modelNotChanged(err) {
		return httpserver.NewStatusResponse(http.StatusNotModified), nil
	}

	if err != nil {
		return nil, err
	}

	err = repo.Update(ctx, model)

	if db.IsDuplicateEntryError(err) {
		return httpserver.NewStatusResponse(http.StatusConflict), nil
	}

	if errors.Is(err, &validation.Error{}) {
		return httpserver.GetErrorHandler()(http.StatusBadRequest, err), nil
	}

	if err != nil {
		return nil, err
	}

	reload, err := repo.Read(ctx, *model.GetId())
	if err != nil {
		return nil, err
	}

	apiView := GetApiViewFromHeader(request.Header)
	out, err := uh.transformer.TransformOutput(ctx, reload, apiView)
	if err != nil {
		return nil, err
	}

	return httpserver.NewJsonResponse(out), nil
}

func modelNotChanged(err error) bool {
	return errors.Is(err, ErrModelNotChanged)
}
