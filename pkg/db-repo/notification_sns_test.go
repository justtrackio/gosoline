package db_repo_test

import (
	"context"
	"testing"
	"time"

	db_repo "github.com/justtrackio/gosoline/pkg/db-repo"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/stream"
	streamMocks "github.com/justtrackio/gosoline/pkg/stream/mocks"
	"github.com/stretchr/testify/assert"
)

func Test_SNS_Notifier(t *testing.T) {
	input := &modelBased{
		value: "my test input",
	}
	modelId := mdl.ModelId{
		Project:     "testProject",
		Name:        "myTest",
		Application: "testApp",
		Family:      "testFamily",
		Environment: "test",
	}
	streamMessage := stream.Message{
		Attributes: map[string]interface{}{
			"encoding": stream.EncodingJson,
			"modelId":  modelId.String(),
			"type":     "CREATE",
			"version":  55,
		},
		Body: `{}`,
	}
	output := streamMocks.Output{}
	output.On("WriteOne", context.Background(), &streamMessage).Return(nil).Once()
	transformer := func(view string, version int, in interface{}) (out interface{}) {
		assert.Equal(t, mdl.Uint(3), in.(db_repo.ModelBased).GetId())
		assert.Equal(t, "api", view)
		assert.Equal(t, 55, version)

		return in
	}

	notifier := db_repo.NewStreamNotifier(logMocks.NewLoggerMockedAll(), &output, modelId, 55, transformer)

	err := notifier.Send(context.Background(), "CREATE", input)
	assert.NoError(t, err)

	output.AssertExpectations(t)
}

type modelBased struct {
	value                string
	createdAt, updatedAt *time.Time
}

func (m *modelBased) GetId() *uint {
	return mdl.Uint(3)
}

func (m *modelBased) SetUpdatedAt(updatedAt *time.Time) {
	m.updatedAt = updatedAt
}

func (m *modelBased) SetCreatedAt(createdAt *time.Time) {
	m.createdAt = createdAt
}
