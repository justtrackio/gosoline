package stream_test

import (
	"fmt"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cloud/aws/sns/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_snsOutput_WriteOne(t *testing.T) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))

	messages := []*stream.Message{
		mkTestMessage(t, 1, make(map[string]string)),
		mkTestMessage(t, 2, make(map[string]string)),
		mkTestMessage(t, 3, make(map[string]string)),
		mkTestMessage(t, 4, make(map[string]string)),
		mkTestMessage(t, 5, make(map[string]string)),
		mkTestMessage(t, 6, make(map[string]string)),
		mkTestMessage(t, 7, make(map[string]string)),
		mkTestMessage(t, 8, make(map[string]string)),
		mkTestMessage(t, 9, make(map[string]string)),
		mkTestMessage(t, 10, make(map[string]string)),
		mkTestMessage(t, 11, make(map[string]string)),
	}

	topic := mocks.NewTopic(t)
	for _, m := range messages {
		//nolint:gocritic // we intentionally avoid %q to prevent double quoting in JSON
		topic.EXPECT().Publish(
			t.Context(),
			fmt.Sprintf(`{"attributes":{"encoding":"application/json"},"body":"%s"}`, m.Body),
			m.Attributes,
		).Return(nil).Once()
	}

	o := stream.NewSnsOutputWithInterfaces(logger, topic)

	for _, val := range messages {
		err := o.WriteOne(t.Context(), val)
		assert.NoError(t, err)
	}
}

func Test_snsOutput_WriteBatch(t *testing.T) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	topic := mocks.NewTopic(t)
	topic.EXPECT().PublishBatch(t.Context(), mock.AnythingOfType("[]string"), mock.AnythingOfType("[]map[string]string")).Return(nil).Once()

	o := stream.NewSnsOutputWithInterfaces(logger, topic)
	batch := []stream.WritableMessage{
		mkTestMessage(t, 1, make(map[string]string)),
		mkTestMessage(t, 2, make(map[string]string)),
		mkTestMessage(t, 3, make(map[string]string)),
		mkTestMessage(t, 4, make(map[string]string)),
		mkTestMessage(t, 5, make(map[string]string)),
		mkTestMessage(t, 6, make(map[string]string)),
		mkTestMessage(t, 7, make(map[string]string)),
		mkTestMessage(t, 8, make(map[string]string)),
		mkTestMessage(t, 9, make(map[string]string)),
		mkTestMessage(t, 10, make(map[string]string)),
		mkTestMessage(t, 11, make(map[string]string)),
	}

	err := o.Write(t.Context(), batch)
	assert.NoError(t, err)
}
