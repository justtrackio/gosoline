package stream_test

import (
	"context"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/sqs"
	sqsMocks "github.com/applike/gosoline/pkg/sqs/mocks"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSqsOutput_WriteOne(t *testing.T) {
	type BodyStruct struct {
		Foo string
	}

	tests := map[string]struct {
		attributes         map[string]interface{}
		body               BodyStruct
		expectedSqsMessage sqs.Message
	}{
		"simple": {
			attributes: map[string]interface{}{},
			body: BodyStruct{
				Foo: "bar",
			},
			expectedSqsMessage: sqs.Message{
				Body: mdl.String(`{"attributes":{"encoding":"application/json"},"body":"{\"Foo\":\"bar\"}"}`),
			},
		},
		"with_delay": {
			attributes: map[string]interface{}{
				sqs.AttributeSqsDelaySeconds: int64(45),
			},
			body: BodyStruct{
				Foo: "bar",
			},
			expectedSqsMessage: sqs.Message{
				DelaySeconds: mdl.Int64(45),
				Body:         mdl.String(`{"attributes":{"encoding":"application/json","sqsDelaySeconds":45},"body":"{\"Foo\":\"bar\"}"}`),
			},
		},
		"with_group_id": {
			attributes: map[string]interface{}{
				sqs.AttributeSqsMessageGroupId: "foo",
			},
			body: BodyStruct{
				Foo: "bar",
			},
			expectedSqsMessage: sqs.Message{
				MessageGroupId: mdl.String("foo"),
				Body:           mdl.String(`{"attributes":{"encoding":"application/json","sqsMessageGroupId":"foo"},"body":"{\"Foo\":\"bar\"}"}`),
			},
		},
		"with_deduplication_id": {
			attributes: map[string]interface{}{
				sqs.AttributeSqsMessageDeduplicationId: "bar",
			},
			body: BodyStruct{
				Foo: "bar",
			},
			expectedSqsMessage: sqs.Message{
				MessageDeduplicationId: mdl.String("bar"),
				Body:                   mdl.String(`{"attributes":{"encoding":"application/json","sqsMessageDeduplicationId":"bar"},"body":"{\"Foo\":\"bar\"}"}`),
			},
		},
		"with_all": {
			attributes: map[string]interface{}{
				sqs.AttributeSqsDelaySeconds:           int64(45),
				sqs.AttributeSqsMessageGroupId:         "foo",
				sqs.AttributeSqsMessageDeduplicationId: "bar",
			},
			body: BodyStruct{
				Foo: "bar",
			},
			expectedSqsMessage: sqs.Message{
				DelaySeconds:           mdl.Int64(45),
				MessageGroupId:         mdl.String("foo"),
				MessageDeduplicationId: mdl.String("bar"),
				Body:                   mdl.String(`{"attributes":{"encoding":"application/json","sqsDelaySeconds":45,"sqsMessageDeduplicationId":"bar","sqsMessageGroupId":"foo"},"body":"{\"Foo\":\"bar\"}"}`),
			},
		},
	}

	for test, data := range tests {
		data := data
		t.Run(test, func(t *testing.T) {
			logger := logMocks.NewLoggerMockedAll()

			queue := new(sqsMocks.Queue)
			queue.On("SendBatch", context.Background(), []*sqs.Message{
				&data.expectedSqsMessage,
			}).Return(nil)

			msg, err := stream.MarshalJsonMessage(data.body, data.attributes)
			assert.NoError(t, err)

			output := stream.NewSqsOutputWithInterfaces(logger, queue, stream.SqsOutputSettings{})
			err = output.WriteOne(context.Background(), msg)

			assert.NoError(t, err)

			queue.AssertExpectations(t)
		})
	}
}
