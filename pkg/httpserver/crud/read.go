package crud

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	dbRepo "github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/pkg/errors"
)

type readHandler[O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]] struct {
	logger      log.Logger
	transformer BaseHandler[O, K, M]
}

func NewReadHandler[O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]](logger log.Logger, transformer BaseHandler[O, K, M]) gin.HandlerFunc {
	rh := readHandler[O, K, M]{
		transformer: transformer,
		logger:      logger,
	}

	return httpserver.CreateHandler(rh)
}

func (rh readHandler[O, K, M]) Handle(ctx context.Context, request *httpserver.Request) (*httpserver.Response, error) {
	id, valid := httpserver.GetIdentifierFromRequest[K](request, "id")

	if !valid {
		return nil, errors.New("no valid id provided")
	}

	repo := rh.transformer.GetRepository()
	model, err := repo.Read(ctx, *id)

	var notFound dbRepo.RecordNotFoundError
	if errors.As(err, &notFound) {
		rh.logger.WithContext(ctx).Warn("failed to read model: %s", err)

		return httpserver.NewStatusResponse(http.StatusNotFound), nil
	}

	if err != nil {
		return nil, err
	}

	apiView := GetApiViewFromHeader(request.Header)
	out, err := rh.transformer.TransformOutput(ctx, model, apiView)
	if err != nil {
		return nil, err
	}

	return httpserver.NewJsonResponse(out), nil
}
