package crud_test

import (
	"context"
	"errors"
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

type patchTestSuite struct {
	suite.Suite

	handler      handler
	patchHandler gin.HandlerFunc
}

func Test_RunPatchTestSuite(t *testing.T) {
	suite.Run(t, new(patchTestSuite))
}

func (s *patchTestSuite) SetupTest() {
	var err error

	config := configMocks.NewConfig(s.T())
	config.EXPECT().UnmarshalKey("crud", mock.AnythingOfType("*crud.Settings")).Run(func(_ string, val any, _ ...cfg.UnmarshalDefaults) {
		settings := val.(*crud.Settings)
		settings.WriteTimeout = time.Minute
	}).Return(nil)

	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))

	s.handler = newHandler(s.T())
	s.patchHandler, err = crud.NewPatchHandler(config, logger, s.handler)
	s.NoError(err)
}

func (s *patchTestSuite) TestPatch() {
	readModel := &Model{}
	updatedModel := &Model{
		Model: db_repo.Model{
			Id: mdl.Box(uint(1)),
			Timestamps: db_repo.Timestamps{
				UpdatedAt: &time.Time{},
				CreatedAt: &time.Time{},
			},
		},
		Name: mdl.Box("updated"),
	}

	s.handler.Repo.EXPECT().Read(matcher.Context, mdl.Box(uint(1)), readModel).
		Run(func(_ context.Context, _ *uint, out db_repo.ModelBased) {
			model := out.(*Model)
			model.Id = mdl.Box(uint(1))
			model.Name = mdl.Box("current")
			model.UpdatedAt = &time.Time{}
			model.CreatedAt = &time.Time{}
		}).Return(nil).Once()
	s.handler.Repo.EXPECT().Update(matcher.Context, updatedModel).Return(nil).Once()
	s.handler.Repo.EXPECT().Read(matcher.Context, mdl.Box(uint(1)), &Model{}).
		Run(func(_ context.Context, _ *uint, out db_repo.ModelBased) {
			model := out.(*Model)
			model.Id = mdl.Box(uint(1))
			model.Name = mdl.Box("updated")
			model.UpdatedAt = &time.Time{}
			model.CreatedAt = &time.Time{}
		}).Return(nil).Once()

	response := httpserver.HttpTest("PATCH", "/:id", "/1", `{"name":"updated"}`, s.patchHandler)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(`{"id":1,"updatedAt":"0001-01-01T00:00:00Z","createdAt":"0001-01-01T00:00:00Z","name":"updated"}`,
		response.Body.String())
}

func (s *patchTestSuite) TestPatch_ValidationError() {
	readModel := &Model{}
	updatedModel := &Model{
		Model: db_repo.Model{
			Id: mdl.Box(uint(1)),
			Timestamps: db_repo.Timestamps{
				UpdatedAt: &time.Time{},
				CreatedAt: &time.Time{},
			},
		},
		Name: mdl.Box("updated"),
	}

	s.handler.Repo.EXPECT().Read(matcher.Context, mdl.Box(uint(1)), readModel).
		Run(func(_ context.Context, _ *uint, out db_repo.ModelBased) {
			model := out.(*Model)
			model.Id = mdl.Box(uint(1))
			model.Name = mdl.Box("current")
			model.UpdatedAt = &time.Time{}
			model.CreatedAt = &time.Time{}
		}).Return(nil).Once()
	s.handler.Repo.EXPECT().Update(matcher.Context, updatedModel).Return(&validation.Error{
		Errors: []error{errors.New("invalid foobar")},
	}).Once()

	response := httpserver.HttpTest("PATCH", "/:id", "/1", `{"name":"updated"}`, s.patchHandler)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"err":"validation: invalid foobar"}`, response.Body.String())
}

func (s *patchTestSuite) TestPatch_InvalidId() {
	response := httpserver.HttpTest("PATCH", "/:id", "/invalid", `{"name":"updated"}`, s.patchHandler)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"err":"validation: no valid id provided"}`, response.Body.String())
}

func (s *patchTestSuite) TestPatch_ReadNotFound() {
	readModel := &Model{}

	s.handler.Repo.EXPECT().Read(matcher.Context, mdl.Box(uint(1)), readModel).
		Return(db_repo.NewRecordNotFoundError(1, "testModel", errors.New("missing"))).Once()

	response := httpserver.HttpTest("PATCH", "/:id", "/1", `{"name":"updated"}`, s.patchHandler)

	s.Equal(http.StatusNotFound, response.Code)
	s.Equal("Not Found", response.Body.String())
}

func (s *patchTestSuite) TestPatch_UnmarshalPatchedModel() {
	readModel := &Model{}

	s.handler.Repo.EXPECT().Read(matcher.Context, mdl.Box(uint(1)), readModel).
		Run(func(_ context.Context, _ *uint, out db_repo.ModelBased) {
			model := out.(*Model)
			model.Id = mdl.Box(uint(1))
			model.Name = mdl.Box("current")
			model.UpdatedAt = &time.Time{}
			model.CreatedAt = &time.Time{}
		}).Return(nil).Once()

	response := httpserver.HttpTest("PATCH", "/:id", "/1", `{"name":123}`, s.patchHandler)

	s.Equal(http.StatusInternalServerError, response.Code)
	s.Contains(response.Body.String(), `failed to unmarshal patched model`)
	s.Contains(response.Body.String(), `cannot unmarshal number`)
}

func (s *patchTestSuite) TestPatch_InvalidBody() {
	response := httpserver.HttpTest("PATCH", "/:id", "/1", `{`, s.patchHandler)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"err":"unexpected EOF"}`, response.Body.String())
}
