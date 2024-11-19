package crud_test

import (
	"net/http"
	"testing"

	configMocks "github.com/justtrackio/gosoline/pkg/cfg/mocks"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/httpserver/crud"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestListHandler_Handle(t *testing.T) {
	config := configMocks.NewConfig(t)
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	transformer := newHandler(t)
	handler := crud.NewListHandler(config, logger, transformer)

	qb := db_repo.NewQueryBuilder()
	qb.Table("footable")
	qb.Where("(((name = ?)))", "foobar")
	qb.GroupBy("id")
	qb.OrderBy("name", "ASC")
	qb.Page(0, 2)

	transformer.Repo.EXPECT().GetMetadata().Return(db_repo.Metadata{
		TableName:  "footable",
		PrimaryKey: "id",
		Mappings: db_repo.FieldMappings{
			"id":   db_repo.NewFieldMapping("id"),
			"name": db_repo.NewFieldMapping("name"),
		},
	})
	transformer.Repo.EXPECT().Count(mock.AnythingOfType("context.backgroundCtx"), qb, &Model{}).Return(1, nil)

	body := `{"filter":{"matches":[{"values":["foobar"],"dimension":"name","operator":"="}],"bool":"and"},"order":[{"field":"name","direction":"ASC"}],"page":{"offset":0,"limit":2}}`
	response := httpserver.HttpTest("PUT", "/:id", "/1", body, handler)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"total":1,"results":[{"Id":1,"UpdatedAt":"2006-01-02T15:04:05Z","CreatedAt":"2006-01-02T15:04:05Z","name":"foobar"}]}`, response.Body.String())

	transformer.Repo.AssertExpectations(t)
}
