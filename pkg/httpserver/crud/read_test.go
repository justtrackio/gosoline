package crud_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	configMocks "github.com/justtrackio/gosoline/pkg/cfg/mocks"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/httpserver/crud"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/testify/assert"
)

func TestReadHandler_Handle(t *testing.T) {
	model := &Model{}

	config := configMocks.NewConfig(t)
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	transformer := newHandler(t)
	transformer.Repo.EXPECT().Read(matcher.Context, mdl.Box(uint(1)), model).Run(func(_ context.Context, _ *uint, out db_repo.ModelBased) {
		model := out.(*Model)
		model.Id = mdl.Box(uint(1))
		model.Name = mdl.Box("foobar")
		model.UpdatedAt = &time.Time{}
		model.CreatedAt = &time.Time{}
	}).Return(nil)

	handler := crud.NewReadHandler(config, logger, transformer)

	response := httpserver.HttpTest("GET", "/:id", "/1", "", handler)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"id":1,"updatedAt":"0001-01-01T00:00:00Z","createdAt":"0001-01-01T00:00:00Z","name":"foobar"}`, response.Body.String())

	transformer.Repo.AssertExpectations(t)
}
