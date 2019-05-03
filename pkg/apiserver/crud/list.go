package crud

import (
	"context"
	"github.com/applike/gosoline/pkg/apiserver"
	"github.com/gin-gonic/gin"
)

type Order struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

type Page struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

type Filter struct {
	Matches []FilterMatch `json:"matches"`
	Groups  []Filter      `json:"groups"`
	Bool    string        `json:"bool"`
}

type FilterMatch struct {
	Dimension string        `json:"dimension"`
	Operator  string        `json:"operator"`
	Values    []interface{} `json:"values"`
}

type Input struct {
	Filter Filter  `json:"filter"`
	Order  []Order `json:"order"`
	Page   *Page   `json:"page"`
}

type Output struct {
	Total   int         `json:"total"`
	Results interface{} `json:"results"`
}

type listHandler struct {
	transformer Handler
}

func NewListHandler(transformer Handler) gin.HandlerFunc {
	lh := listHandler{
		transformer: transformer,
	}

	return apiserver.CreateJsonHandler(lh)
}

func (lh listHandler) GetInput() interface{} {
	return &Input{
		Order: make([]Order, 0),
	}
}

func (lh listHandler) Handle(ctx context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	inp := request.Body.(*Input)

	repo := lh.transformer.GetRepository()
	metadata := repo.GetMetadata()

	lqb := NewListQueryBuilder(metadata)
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
