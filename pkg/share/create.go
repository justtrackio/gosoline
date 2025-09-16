package share

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/httpserver/crud"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/uuid"
	"github.com/justtrackio/gosoline/pkg/validation"
)

type shareCreateHandler[I Metadata, K mdl.PossibleIdentifier, M Shareable[K]] struct {
	logger       log.Logger
	transformer  ShareCreateHandler[I, K, M]
	uuidProvider uuid.Uuid
}

func NewShareCreateHandler[I Metadata, K mdl.PossibleIdentifier, M Shareable[K]](logger log.Logger, transformer ShareCreateHandler[I, K, M]) gin.HandlerFunc {
	sh := shareCreateHandler[I, K, M]{
		logger:       logger,
		transformer:  transformer,
		uuidProvider: uuid.New(),
	}

	return httpserver.CreateJsonHandler(sh)
}

func (s shareCreateHandler[I, K, M]) GetInput() any {
	var input I

	return &input
}

func (s shareCreateHandler[I, K, M]) Handle(ctx context.Context, req *httpserver.Request) (*httpserver.Response, error) {
	id, valid := httpserver.GetIdentifierFromRequest[K](req, "id")

	if !valid {
		return nil, errors.New("no valid id provided")
	}

	entity, err := s.getEntity(ctx, *id)
	var notFound db_repo.RecordNotFoundError
	if errors.As(err, &notFound) {
		s.logger.Warn(ctx, "failed to read entity: %s", err.Error())

		return httpserver.NewStatusResponse(http.StatusNotFound), nil
	}

	if err != nil {
		return nil, err
	}

	// we assert cast safely here as the req.Body will get parsed in something that implements Metadata
	shareInput := *req.Body.(*I)
	policy := BuildSharePolicy[K](s.uuidProvider.NewV4(), entity, shareInput.GetOwnerId(), shareInput.GetActions())

	guard := s.transformer.GetGuard()
	err = guard.CreatePolicy(ctx, policy)
	if err != nil {
		return nil, err
	}

	model, err := s.transformer.TransformCreate(ctx, shareInput, entity, policy)
	if err != nil {
		return nil, err
	}

	shareRepo := s.transformer.GetRepository()
	err = shareRepo.Create(ctx, model)

	if db.IsDuplicateEntryError(err) {
		return httpserver.NewStatusResponse(http.StatusConflict), nil
	}

	if errors.Is(err, &validation.Error{}) {
		return httpserver.GetErrorHandler()(http.StatusBadRequest, err), nil
	}

	if err != nil {
		return nil, err
	}

	reload, err := shareRepo.Read(ctx, *model.GetId())
	if err != nil {
		return nil, err
	}

	apiView := crud.GetApiViewFromHeader(req.Header)
	out, err := s.transformer.TransformOutput(ctx, reload, apiView)
	if err != nil {
		return nil, err
	}

	return httpserver.NewJsonResponse(out), nil
}

func (s shareCreateHandler[I, K, M]) getEntity(ctx context.Context, id K) (M, error) {
	entityRepo := s.transformer.GetEntityRepository()

	return entityRepo.Read(ctx, id)
}
