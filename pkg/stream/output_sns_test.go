package stream_test

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cloud/aws/sns/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_snsOutput_Write(t *testing.T) {
	logger := logMocks.NewLoggerMockedAll()
	topic := &mocks.Topic{}
	topic.On("PublishBatch", context.Background(), mock.AnythingOfType("[]string"), mock.AnythingOfType("[]map[string]interface {}")).Return(nil).Once()

	o := stream.NewSnsOutputWithInterfaces(logger, topic)
	batch := []stream.WritableMessage{
		mkTestMessage(t, 1, make(map[string]interface{})),
		mkTestMessage(t, 2, make(map[string]interface{})),
		mkTestMessage(t, 3, make(map[string]interface{})),
		mkTestMessage(t, 4, make(map[string]interface{})),
		mkTestMessage(t, 5, make(map[string]interface{})),
		mkTestMessage(t, 6, make(map[string]interface{})),
		mkTestMessage(t, 7, make(map[string]interface{})),
		mkTestMessage(t, 8, make(map[string]interface{})),
		mkTestMessage(t, 9, make(map[string]interface{})),
		mkTestMessage(t, 10, make(map[string]interface{})),
		mkTestMessage(t, 11, make(map[string]interface{})),
	}

	err := o.Write(context.Background(), batch)
	assert.NoError(t, err)

	topic.AssertExpectations(t)
}
