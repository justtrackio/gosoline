package stream_test

import (
	"context"
	"github.com/applike/gosoline/pkg/mdl"
	monMocks "github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/applike/gosoline/pkg/sqs"
	sqsMocks "github.com/applike/gosoline/pkg/sqs/mocks"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestSqsOutput_WriteOne(t *testing.T) {
	logger := monMocks.NewLoggerMockedAll()
	tracer := tracing.NewNoopTracer()

	expectedBody := `{"trace":{"traceId":"","id":"","parentId":"","sampled":false},"attributes":{"sqsDelaySeconds":45},"body":"{\"Foo\":\"bar\"}"}`
	expectedSqsMessages := []*sqs.Message{
		{
			DelaySeconds: mdl.Int64(45),
			Body:         mdl.String(expectedBody),
		},
	}

	queue := new(sqsMocks.Queue)
	queue.On("SendBatch", mock.AnythingOfType("*context.emptyCtx"), expectedSqsMessages).Return(nil)

	msg, err := BuildSqsTestMessage()
	assert.NoError(t, err)

	output := stream.NewSqsOutputWithInterfaces(logger, tracer, queue, stream.SqsOutputSettings{})
	err = output.WriteOne(context.Background(), msg)

	assert.NoError(t, err)
}
