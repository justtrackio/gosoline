package crud_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/cfg"
	configMocks "github.com/justtrackio/gosoline/pkg/cfg/mocks"
	dbRepo "github.com/justtrackio/gosoline/pkg/db-repo"
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
	var err error

	config := configMocks.NewConfig(s.T())
	config.EXPECT().UnmarshalKey("crud", mock.AnythingOfType("*crud.Settings")).Run(func(key string, val any, additionalDefaults ...cfg.UnmarshalDefaults) {
		settings := val.(*crud.Settings)
		settings.WriteTimeout = time.Minute
	}).Return(nil).Once()

	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))

	s.handler = newHandler(s.T())
	s.updateHandler, err = crud.NewUpdateHandler[UpdateInput, Output, uint, *Model](config, logger, s.handler)
	s.NoError(err)
}

func (s *updateTestSuite) TestUpdate() {
	updateModel := &Model{
		Model: dbRepo.Model{
			Id: mdl.Box(uint(1)),
			Timestamps: dbRepo.Timestamps{
				UpdatedAt: &time.Time{},
				CreatedAt: &time.Time{},
			},
		},
		Name: mdl.Box("updated"),
	}

	s.handler.Repo.EXPECT().Update(matcher.Context, updateModel).Return(nil).Once()
	s.handler.Repo.EXPECT().Read(matcher.Context, uint(1)).Return(&Model{
		Model: dbRepo.Model{
			Id: mdl.Box(uint(1)),
			Timestamps: dbRepo.Timestamps{
				UpdatedAt: &time.Time{},
				CreatedAt: &time.Time{},
			},
		},
		Name: mdl.Box("initial"),
	}, nil).Once()
	s.handler.Repo.EXPECT().Read(matcher.Context, uint(1)).Return(&Model{
		Model: dbRepo.Model{
			Id: mdl.Box(uint(1)),
			Timestamps: dbRepo.Timestamps{
				UpdatedAt: &time.Time{},
				CreatedAt: &time.Time{},
			},
		},
		Name: mdl.Box("updated"),
	}, nil).Once()

	body := `{"name": "updated"}`
	response := httpserver.HttpTest("PUT", "/:id", "/1", body, s.updateHandler)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(`{"id":1,"updatedAt":"0001-01-01T00:00:00Z","createdAt":"0001-01-01T00:00:00Z","name":"updated"}`, response.Body.String())
}

func (s *updateTestSuite) TestUpdate_ValidationError() {
	updateModel := &Model{
		Model: dbRepo.Model{
			Id: mdl.Box(uint(1)),
			Timestamps: dbRepo.Timestamps{
				UpdatedAt: &time.Time{},
				CreatedAt: &time.Time{},
			},
		},
		Name: mdl.Box("updated"),
	}

	s.handler.Repo.EXPECT().Update(matcher.Context, updateModel).Return(&validation.Error{
		Errors: []error{fmt.Errorf("invalid foobar")},
	}).Once()
	s.handler.Repo.EXPECT().Read(matcher.Context, uint(1)).Return(&Model{
		Model: dbRepo.Model{
			Id: mdl.Box(uint(1)),
			Timestamps: dbRepo.Timestamps{
				UpdatedAt: &time.Time{},
				CreatedAt: &time.Time{},
			},
		},
		Name: mdl.Box("updated"),
	}, nil).Once()

	body := `{"name": "updated"}`
	response := httpserver.HttpTest("PUT", "/:id", "/1", body, s.updateHandler)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"err":"validation: invalid foobar"}`, response.Body.String())
}
