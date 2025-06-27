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

type updateTestSuite struct {
	suite.Suite

	handler       handler
	updateHandler gin.HandlerFunc
}

func Test_RunUpdateTestSuite(t *testing.T) {
	suite.Run(t, new(updateTestSuite))
}

func (s *updateTestSuite) SetupTest() {
	config := configMocks.NewConfig(s.T())
	config.EXPECT().UnmarshalKey("crud", mock.AnythingOfType("*crud.Settings")).Run(func(key string, val any, additionalDefaults ...cfg.UnmarshalDefaults) {
		settings := val.(*crud.Settings)
		settings.WriteTimeout = time.Minute
	})

	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))

	s.handler = newHandler(s.T())
	s.updateHandler = crud.NewUpdateHandler(config, logger, s.handler)
}

func (s *updateTestSuite) TestUpdate() {
	readModel := &Model{}
	updateModel := &Model{
		Model: db_repo.Model{
			Id: mdl.Box(uint(1)),
			Timestamps: db_repo.Timestamps{
				UpdatedAt: &time.Time{},
				CreatedAt: &time.Time{},
			},
		},
		Name: mdl.Box("updated"),
	}

	s.handler.Repo.EXPECT().Update(matcher.Context, updateModel).Return(nil)
	s.handler.Repo.EXPECT().Read(matcher.Context, mdl.Box(uint(1)), readModel).
		Run(func(ctx context.Context, id *uint, out db_repo.ModelBased) {
			model := out.(*Model)
			model.Id = mdl.Box(uint(1))
			model.Name = mdl.Box("updated")
			model.UpdatedAt = &time.Time{}
			model.CreatedAt = &time.Time{}
		}).Return(nil)

	body := `{"name": "updated"}`
	response := httpserver.HttpTest("PUT", "/:id", "/1", body, s.updateHandler)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(`{"id":1,"updatedAt":"0001-01-01T00:00:00Z","createdAt":"0001-01-01T00:00:00Z","name":"updated"}`, response.Body.String())
}

func (s *updateTestSuite) TestUpdate_ValidationError() {
	readModel := &Model{}
	updateModel := &Model{
		Model: db_repo.Model{
			Id: mdl.Box(uint(1)),
			Timestamps: db_repo.Timestamps{
				UpdatedAt: &time.Time{},
				CreatedAt: &time.Time{},
			},
		},
		Name: mdl.Box("updated"),
	}

	s.handler.Repo.EXPECT().Update(matcher.Context, updateModel).Return(&validation.Error{
		Errors: []error{fmt.Errorf("invalid foobar")},
	})
	s.handler.Repo.EXPECT().Read(matcher.Context, mdl.Box(uint(1)), readModel).Run(func(_ context.Context, _ *uint, out db_repo.ModelBased) {
		model := out.(*Model)
		model.Id = mdl.Box(uint(1))
		model.Name = mdl.Box("updated")
		model.UpdatedAt = &time.Time{}
		model.CreatedAt = &time.Time{}
	}).Return(nil)

	body := `{"name": "updated"}`
	response := httpserver.HttpTest("PUT", "/:id", "/1", body, s.updateHandler)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"err":"validation: invalid foobar"}`, response.Body.String())
}
