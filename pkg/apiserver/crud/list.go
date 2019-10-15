package crud

import (
	"context"
	"github.com/applike/gosoline/pkg/apiserver"
	"github.com/applike/gosoline/pkg/apiserver/sql"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/gin-gonic/gin"
)

type Output struct {
	Total   int         `json:"total"`
	Results interface{} `json:"results"`
}

type listHandler struct {
	transformer Handler
}

func NewListHandler(transformer Handler, config cfg.Config, logger mon.Logger) gin.HandlerFunc {
	lh := listHandler{
		transformer: transformer,
	}

	return apiserver.CreateJsonHandler(lh, config, logger)
}

func (lh listHandler) GetInput() interface{} {
	return sql.NewInput()
}

func (lh listHandler) Handle(ctx context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	inp := request.Body.(*sql.Input)

	repo := lh.transformer.GetRepository()
	metadata := repo.GetMetadata()

	lqb := sql.NewOrmQueryBuilder(metadata)
	qb, err := lqb.Build(inp)

	if err != nil {
		return nil, err
	}

	apiView := getApiViewFromHeader(request.Header)
	results, err := lh.transformer.List(ctx, qb, apiView)

	if err != nil {
		return nil, err
	}

	model := lh.transformer.GetModel()
	total, err := repo.Count(ctx, qb, model)

	if err != nil {
		return nil, err
	}

	out := Output{
		Total:   total,
		Results: results,
	}

	resp := apiserver.NewJsonResponse(out)
	resp.AddHeader(apiserver.ApiViewKey, apiView)

	return resp, nil
}
