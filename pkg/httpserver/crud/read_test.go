package crud_test

import (
	"net/http"
	"testing"
	"time"

	dbRepo "github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/httpserver/crud"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/testify/assert"
)

func TestReadHandler_Handle(t *testing.T) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	transformer := newHandler(t)
	transformer.Repo.EXPECT().Read(matcher.Context, uint(1)).Return(&Model{
		Model: dbRepo.Model{
			Id: mdl.Box(uint(1)),
			Timestamps: dbRepo.Timestamps{
				UpdatedAt: &time.Time{},
				CreatedAt: &time.Time{},
			},
		},
		Name: mdl.Box("foobar"),
	}, nil).Once()

	handler := crud.NewReadHandler[Output, uint, *Model](logger, transformer)

	response := httpserver.HttpTest("GET", "/:id", "/1", "", handler)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"id":1,"updatedAt":"0001-01-01T00:00:00Z","createdAt":"0001-01-01T00:00:00Z","name":"foobar"}`, response.Body.String())
}
