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

type createHandler[I any, O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]] struct {
	logger      log.Logger
	transformer CreateHandler[I, O, K, M]
}

func NewCreateHandler[I any, O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]](logger log.Logger, transformer CreateHandler[I, O, K, M]) gin.HandlerFunc {
	ch := createHandler[I, O, K, M]{
		transformer: transformer,
		logger:      logger,
	}

	return httpserver.CreateJsonHandler(ch)
}

func (ch createHandler[I, O, K, M]) GetInput() any {
	var input I

	return &input
}

func (ch createHandler[I, O, K, M]) Handle(ctx context.Context, request *httpserver.Request) (*httpserver.Response, error) {
	model, err := ch.transformer.TransformCreate(ctx, request.Body.(*I))
	if err != nil {
		return nil, err
	}

	repo := ch.transformer.GetRepository()
	err = repo.Create(ctx, model)

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
	out, err := ch.transformer.TransformOutput(ctx, reload, apiView)
	if err != nil {
		return nil, err
	}

	return httpserver.NewJsonResponse(out), nil
}
