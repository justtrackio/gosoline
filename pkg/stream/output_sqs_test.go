package stream_test

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cloud/aws/sqs"
	sqsMocks "github.com/justtrackio/gosoline/pkg/cloud/aws/sqs/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/stretchr/testify/assert"
	"github.com/thoas/go-funk"
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
				sqs.AttributeSqsDelaySeconds: int32(45),
			},
			body: BodyStruct{
				Foo: "bar",
			},
			expectedSqsMessage: sqs.Message{
				DelaySeconds: 45,
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
				sqs.AttributeSqsDelaySeconds:           int32(45),
				sqs.AttributeSqsMessageGroupId:         "foo",
				sqs.AttributeSqsMessageDeduplicationId: "bar",
			},
			body: BodyStruct{
				Foo: "bar",
			},
			expectedSqsMessage: sqs.Message{
				DelaySeconds:           45,
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
			queue.On("Send", context.Background(), &data.expectedSqsMessage).Return(nil).Once()

			msg, err := stream.MarshalJsonMessage(data.body, data.attributes)
			assert.NoError(t, err)

			output := stream.NewSqsOutputWithInterfaces(logger, queue, &stream.SqsOutputSettings{})
			err = output.WriteOne(context.Background(), msg)

			assert.NoError(t, err)
			queue.AssertExpectations(t)
		})
	}
}

func TestSqsOutput_Write(t *testing.T) {
	type BodyStruct struct {
		Foo string
	}

	largeAttributes, err := funk.Fill(make([]map[string]interface{}, 1000), map[string]interface{}{sqs.AttributeSqsMessageGroupId: "foo"})
	assert.NoError(t, err)

	largeBody, err := funk.Fill(make([]BodyStruct, 1000), BodyStruct{Foo: "bar"})
	assert.NoError(t, err)

	largeExpectedSqsMessage, err := funk.Fill(make([]*sqs.Message, 1000), &sqs.Message{
		MessageGroupId: mdl.String("foo"),
		Body:           mdl.String(`{"attributes":{"encoding":"application/json","sqsMessageGroupId":"foo"},"body":"{\"Foo\":\"bar\"}"}`),
	})
	assert.NoError(t, err)

	tests := map[string]struct {
		attributes         []map[string]interface{}
		body               []BodyStruct
		expectedSqsMessage []*sqs.Message
	}{
		"single": {
			attributes: []map[string]interface{}{
				{},
			},
			body: []BodyStruct{
				{Foo: "bar"},
			},
			expectedSqsMessage: []*sqs.Message{
				{
					Body: mdl.String(`{"attributes":{"encoding":"application/json"},"body":"{\"Foo\":\"bar\"}"}`),
				},
			},
		},
		"multiple": {
			attributes: []map[string]interface{}{
				{sqs.AttributeSqsDelaySeconds: int32(45)},
				{sqs.AttributeSqsMessageGroupId: "foo"},
				{sqs.AttributeSqsMessageGroupId: "foo1"},
				{sqs.AttributeSqsMessageGroupId: "foo2"},
				{sqs.AttributeSqsMessageGroupId: "foo3"},
				{sqs.AttributeSqsMessageGroupId: "foo4"},
				{sqs.AttributeSqsMessageGroupId: "bar"},
				{sqs.AttributeSqsMessageDeduplicationId: "bar"},
				{
					sqs.AttributeSqsDelaySeconds:           int32(45),
					sqs.AttributeSqsMessageGroupId:         "foo1",
					sqs.AttributeSqsMessageDeduplicationId: "bar1",
				},
				{
					sqs.AttributeSqsDelaySeconds:           int32(46),
					sqs.AttributeSqsMessageGroupId:         "foo2",
					sqs.AttributeSqsMessageDeduplicationId: "bar2",
				},
				{
					sqs.AttributeSqsDelaySeconds:           int32(47),
					sqs.AttributeSqsMessageGroupId:         "foo3",
					sqs.AttributeSqsMessageDeduplicationId: "bar3",
				},
				{
					sqs.AttributeSqsDelaySeconds:           int32(48),
					sqs.AttributeSqsMessageGroupId:         "foo4",
					sqs.AttributeSqsMessageDeduplicationId: "bar4",
				},
			},
			body: []BodyStruct{
				{Foo: "bar"},
				{Foo: "bar"},
				{Foo: "bar1"},
				{Foo: "bar2"},
				{Foo: "bar3"},
				{Foo: "bar4"},
				{Foo: "bar"},
				{Foo: "bar"},
				{Foo: "multipleAttributes1"},
				{Foo: "multipleAttributes2"},
				{Foo: "multipleAttributes3"},
				{Foo: "multipleAttributes4"},
			},
			expectedSqsMessage: []*sqs.Message{
				{
					DelaySeconds: 45,
					Body:         mdl.String(`{"attributes":{"encoding":"application/json","sqsDelaySeconds":45},"body":"{\"Foo\":\"bar\"}"}`),
				},
				{
					MessageGroupId: mdl.String("foo"),
					Body:           mdl.String(`{"attributes":{"encoding":"application/json","sqsMessageGroupId":"foo"},"body":"{\"Foo\":\"bar\"}"}`),
				},
				{
					MessageGroupId: mdl.String("foo1"),
					Body:           mdl.String(`{"attributes":{"encoding":"application/json","sqsMessageGroupId":"foo1"},"body":"{\"Foo\":\"bar1\"}"}`),
				},
				{
					MessageGroupId: mdl.String("foo2"),
					Body:           mdl.String(`{"attributes":{"encoding":"application/json","sqsMessageGroupId":"foo2"},"body":"{\"Foo\":\"bar2\"}"}`),
				},
				{
					MessageGroupId: mdl.String("foo3"),
					Body:           mdl.String(`{"attributes":{"encoding":"application/json","sqsMessageGroupId":"foo3"},"body":"{\"Foo\":\"bar3\"}"}`),
				},
				{
					MessageGroupId: mdl.String("foo4"),
					Body:           mdl.String(`{"attributes":{"encoding":"application/json","sqsMessageGroupId":"foo4"},"body":"{\"Foo\":\"bar4\"}"}`),
				},
				{
					MessageGroupId: mdl.String("bar"),
					Body:           mdl.String(`{"attributes":{"encoding":"application/json","sqsMessageGroupId":"bar"},"body":"{\"Foo\":\"bar\"}"}`),
				},
				{
					MessageDeduplicationId: mdl.String("bar"),
					Body:                   mdl.String(`{"attributes":{"encoding":"application/json","sqsMessageDeduplicationId":"bar"},"body":"{\"Foo\":\"bar\"}"}`),
				},
				{
					DelaySeconds:           45,
					MessageGroupId:         mdl.String("foo1"),
					MessageDeduplicationId: mdl.String("bar1"),
					Body:                   mdl.String(`{"attributes":{"encoding":"application/json","sqsDelaySeconds":45,"sqsMessageDeduplicationId":"bar1","sqsMessageGroupId":"foo1"},"body":"{\"Foo\":\"multipleAttributes1\"}"}`),
				},
				{
					DelaySeconds:           46,
					MessageGroupId:         mdl.String("foo2"),
					MessageDeduplicationId: mdl.String("bar2"),
					Body:                   mdl.String(`{"attributes":{"encoding":"application/json","sqsDelaySeconds":46,"sqsMessageDeduplicationId":"bar2","sqsMessageGroupId":"foo2"},"body":"{\"Foo\":\"multipleAttributes2\"}"}`),
				},
				{
					DelaySeconds:           47,
					MessageGroupId:         mdl.String("foo3"),
					MessageDeduplicationId: mdl.String("bar3"),
					Body:                   mdl.String(`{"attributes":{"encoding":"application/json","sqsDelaySeconds":47,"sqsMessageDeduplicationId":"bar3","sqsMessageGroupId":"foo3"},"body":"{\"Foo\":\"multipleAttributes3\"}"}`),
				},
				{
					DelaySeconds:           48,
					MessageGroupId:         mdl.String("foo4"),
					MessageDeduplicationId: mdl.String("bar4"),
					Body:                   mdl.String(`{"attributes":{"encoding":"application/json","sqsDelaySeconds":48,"sqsMessageDeduplicationId":"bar4","sqsMessageGroupId":"foo4"},"body":"{\"Foo\":\"multipleAttributes4\"}"}`),
				},
			},
		},
		"large": {
			attributes:         largeAttributes.([]map[string]interface{}),
			body:               largeBody.([]BodyStruct),
			expectedSqsMessage: largeExpectedSqsMessage.([]*sqs.Message),
		},
	}

	for test, data := range tests {
		data := data
		t.Run(test, func(t *testing.T) {
			logger := logMocks.NewLoggerMockedAll()

			expectedSqsMessageChunks, ok := funk.Chunk(data.expectedSqsMessage, stream.SqsOutputBatchSize).([][]*sqs.Message)
			assert.True(t, ok)

			queue := new(sqsMocks.Queue)
			for _, chunk := range expectedSqsMessageChunks {
				queue.On("SendBatch", context.Background(), chunk).Return(nil).Once()
			}

			messages := make([]stream.WritableMessage, len(data.body))
			for i := range data.body {
				var err error
				messages[i], err = stream.MarshalJsonMessage(data.body[i], data.attributes[i])

				assert.NoError(t, err)
			}

			output := stream.NewSqsOutputWithInterfaces(logger, queue, &stream.SqsOutputSettings{})
			err := output.Write(context.Background(), messages)

			assert.NoError(t, err)
			queue.AssertExpectations(t)
		})
	}
}
