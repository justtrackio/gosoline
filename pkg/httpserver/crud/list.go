package crud

import (
	"context"

	"github.com/gin-gonic/gin"
	dbRepo "github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/httpserver/sql"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/validation"
)

type Output[O any] struct {
	Total   int `json:"total"`
	Results []O `json:"results"`
}

type listHandler[O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]] struct {
	logger      log.Logger
	transformer ListHandler[O, K, M]
}

func NewListHandler[O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]](logger log.Logger, transformer ListHandler[O, K, M]) gin.HandlerFunc {
	lh := listHandler[O, K, M]{
		logger:      logger,
		transformer: transformer,
	}

	return httpserver.CreateJsonHandler(lh)
}

func (lh listHandler[O, K, M]) GetInput() any {
	return sql.NewInput()
}

func (lh listHandler[O, K, M]) Handle(ctx context.Context, request *httpserver.Request) (*httpserver.Response, error) {
	inp := request.Body.(*sql.Input)

	repo := lh.transformer.GetRepository()
	metadata := repo.GetMetadata()

	lqb := sql.NewOrmQueryBuilder(metadata)
	qb, err := lqb.Build(inp)
	if err != nil {
		return HandleErrorOnRead(ctx, lh.logger, &validation.Error{
			Errors: []error{err},
		})
	}

	apiView := GetApiViewFromHeader(request.Header)
	results, err := lh.transformer.List(ctx, qb, apiView)
	if err != nil {
		return HandleErrorOnRead(ctx, lh.logger, err)
	}

	total, err := repo.Count(ctx, qb)
	if err != nil {
		return HandleErrorOnRead(ctx, lh.logger, err)
	}

	out := Output[O]{
		Total:   total,
		Results: results,
	}

	resp := httpserver.NewJsonResponse(out)
	resp.AddHeader(httpserver.ApiViewKey, apiView)

	return resp, nil
}

func DefaultList[O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]](h BaseHandler[O, K, M], ctx context.Context, qb *dbRepo.QueryBuilder, apiView string) ([]O, error) {
	result, err := h.GetRepository().Query(ctx, qb)
	if err != nil {
		return nil, err
	}

	out := make([]O, len(result))
	for i, res := range result {
		out[i], err = h.TransformOutput(ctx, res, apiView)
		if err != nil {
			return nil, err
		}
	}

	return out, nil
}
