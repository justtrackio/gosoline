package db_repo_test

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/mdl"
	mdlMocks "github.com/justtrackio/gosoline/pkg/mdlsub/mocks"
	"github.com/stretchr/testify/assert"
)

type modelBased struct {
	db_repo.Model
	value string
}

func Test_Publish_Notifier(t *testing.T) {
	input := &modelBased{
		Model: db_repo.Model{
			Id: mdl.Box(uint(3)),
		},
		value: "my test input",
	}

	transformer := func(view string, version int, in interface{}) (out interface{}) {
		assert.Equal(t, mdl.Box(uint(3)), in.(db_repo.ModelBased).GetId())
		assert.Equal(t, "api", view)
		assert.Equal(t, 1, version)

		return in
	}

	publisher := mdlMocks.Publisher{}
	publisher.On("Publish", context.Background(), "CREATE", 1, input).Return(nil).Once()

	modelId := mdl.ModelId{
		Project:     "testProject",
		Name:        "myTest",
		Application: "testApp",
		Family:      "testFamily",
		Group:       "grp",
		Environment: "test",
	}

	notifier := db_repo.NewPublisherNotifier(
		context.Background(),
		cfg.New(),
		&publisher,
		logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t)),
		modelId,
		1,
		transformer,
	)

	err := notifier.Send(context.Background(), "CREATE", input)
	assert.NoError(t, err)

	publisher.AssertExpectations(t)
}
