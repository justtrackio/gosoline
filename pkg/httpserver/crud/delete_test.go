package crud_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/cfg"
	configMocks "github.com/justtrackio/gosoline/pkg/cfg/mocks"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/httpserver/crud"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/justtrackio/gosoline/pkg/validation"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type deleteTestSuite struct {
	suite.Suite

	handler       handler
	deleteHandler gin.HandlerFunc
}

func Test_RunDeleteTestSuite(t *testing.T) {
	suite.Run(t, new(deleteTestSuite))
}

func (s *deleteTestSuite) SetupTest() {
	var err error

	config := configMocks.NewConfig(s.T())
	config.EXPECT().UnmarshalKey("crud", mock.AnythingOfType("*crud.Settings")).Run(func(key string, val any, additionalDefaults ...cfg.UnmarshalDefaults) {
		settings := val.(*crud.Settings)
		settings.WriteTimeout = time.Minute
	}).Return(nil)

	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))

	s.handler = newHandler(s.T())
	s.deleteHandler, err = crud.NewDeleteHandler(config, logger, s.handler)
	s.NoError(err)
}

func (s *deleteTestSuite) TestDelete() {
	model := &Model{}
	deleteModel := &Model{
		Model: db_repo.Model{
			Id: mdl.Box(uint(1)),
			Timestamps: db_repo.Timestamps{
				UpdatedAt: &time.Time{},
				CreatedAt: &time.Time{},
			},
		},
		Name: mdl.Box("foobar"),
	}

	s.handler.Repo.EXPECT().Read(matcher.Context, mock.AnythingOfType("*uint"), model).
		Run(func(ctx context.Context, id *uint, out db_repo.ModelBased) {
			model := out.(*Model)
			model.Id = mdl.Box(uint(1))
			model.Name = mdl.Box("foobar")
			model.UpdatedAt = &time.Time{}
			model.CreatedAt = &time.Time{}
		}).Return(nil)
	s.handler.Repo.EXPECT().Delete(matcher.Context, deleteModel).Return(nil)

	response := httpserver.HttpTest("DELETE", "/:id", "/1", "", s.deleteHandler)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(`{"id":1,"updatedAt":"0001-01-01T00:00:00Z","createdAt":"0001-01-01T00:00:00Z","name":"foobar"}`, response.Body.String())
}

func (s *deleteTestSuite) TestDelete_ValidationError() {
	model := &Model{}
	deleteModel := &Model{
		Model: db_repo.Model{
			Id: mdl.Box(uint(1)),
			Timestamps: db_repo.Timestamps{
				UpdatedAt: &time.Time{},
				CreatedAt: &time.Time{},
			},
		},
		Name: mdl.Box("foobar"),
	}

	s.handler.Repo.EXPECT().Read(matcher.Context, mock.AnythingOfType("*uint"), model).
		Run(func(ctx context.Context, id *uint, out db_repo.ModelBased) {
			model := out.(*Model)
			model.Id = mdl.Box(uint(1))
			model.Name = mdl.Box("foobar")
			model.UpdatedAt = &time.Time{}
			model.CreatedAt = &time.Time{}
		}).Return(nil)
	s.handler.Repo.EXPECT().Delete(matcher.Context, deleteModel).Return(&validation.Error{
		Errors: []error{fmt.Errorf("invalid foobar")},
	})

	response := httpserver.HttpTest("DELETE", "/:id", "/1", "", s.deleteHandler)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"err":"validation: invalid foobar"}`, response.Body.String())
}
