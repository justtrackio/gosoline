package crud

import (
	"context"

	"github.com/gin-gonic/gin"
	dbRepo "github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/validation"
	"github.com/pkg/errors"
)

type readHandler[O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]] struct {
	logger      log.Logger
	transformer BaseHandler[O, K, M]
}

func NewReadHandler[O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]](logger log.Logger, transformer BaseHandler[O, K, M]) gin.HandlerFunc {
	rh := readHandler[O, K, M]{
		logger:      logger,
		transformer: transformer,
	}

	return httpserver.CreateHandler(rh)
}

func (rh readHandler[O, K, M]) Handle(ctx context.Context, request *httpserver.Request) (*httpserver.Response, error) {
	id, valid := httpserver.GetIdentifierFromRequest[K](request, "id")

	if !valid {
		return HandleErrorOnRead(ctx, rh.logger, &validation.Error{
			Errors: []error{
				errors.New("no valid id provided"),
			},
		})
	}

	logger := rh.logger.WithFields(log.Fields{
		"entity_id": id,
	})

	repo := rh.transformer.GetRepository()
	model, err := repo.Read(ctx, *id)
	if err != nil {
		return HandleErrorOnRead(ctx, logger, err)
	}

	apiView := GetApiViewFromHeader(request.Header)
	out, err := rh.transformer.TransformOutput(ctx, model, apiView)
	if err != nil {
		return HandleErrorOnRead(ctx, logger, err)
	}

	return httpserver.NewJsonResponse(out), nil
}
