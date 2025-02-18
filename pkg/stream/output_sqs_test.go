package stream_test

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cloud/aws/sqs"
	sqsMocks "github.com/justtrackio/gosoline/pkg/cloud/aws/sqs/mocks"
	"github.com/justtrackio/gosoline/pkg/funk"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/stretchr/testify/assert"
)

func TestSqsOutput_WriteOne(t *testing.T) {
	type BodyStruct struct {
		Foo string
	}

	tests := map[string]struct {
		attributes         map[string]string
		body               BodyStruct
		expectedSqsMessage sqs.Message
	}{
		"simple": {
			attributes: map[string]string{},
			body: BodyStruct{
				Foo: "bar",
			},
			expectedSqsMessage: sqs.Message{
				Body: mdl.Box(`{"attributes":{"encoding":"application/json"},"body":"{\"Foo\":\"bar\"}"}`),
			},
		},
		"with_delay": {
			attributes: map[string]string{
				sqs.AttributeSqsDelaySeconds: "45",
			},
			body: BodyStruct{
				Foo: "bar",
			},
			expectedSqsMessage: sqs.Message{
				DelaySeconds: 45,
				Body:         mdl.Box(`{"attributes":{"encoding":"application/json","sqsDelaySeconds":"45"},"body":"{\"Foo\":\"bar\"}"}`),
			},
		},
		"with_group_id": {
			attributes: map[string]string{
				sqs.AttributeSqsMessageGroupId: "foo",
			},
			body: BodyStruct{
				Foo: "bar",
			},
			expectedSqsMessage: sqs.Message{
				MessageGroupId: mdl.Box("foo"),
				Body:           mdl.Box(`{"attributes":{"encoding":"application/json","sqsMessageGroupId":"foo"},"body":"{\"Foo\":\"bar\"}"}`),
			},
		},
		"with_deduplication_id": {
			attributes: map[string]string{
				sqs.AttributeSqsMessageDeduplicationId: "bar",
			},
			body: BodyStruct{
				Foo: "bar",
			},
			expectedSqsMessage: sqs.Message{
				MessageDeduplicationId: mdl.Box("bar"),
				Body:                   mdl.Box(`{"attributes":{"encoding":"application/json","sqsMessageDeduplicationId":"bar"},"body":"{\"Foo\":\"bar\"}"}`),
			},
		},
		"with_all": {
			attributes: map[string]string{
				sqs.AttributeSqsDelaySeconds:           "45",
				sqs.AttributeSqsMessageGroupId:         "foo",
				sqs.AttributeSqsMessageDeduplicationId: "bar",
			},
			body: BodyStruct{
				Foo: "bar",
			},
			expectedSqsMessage: sqs.Message{
				DelaySeconds:           45,
				MessageGroupId:         mdl.Box("foo"),
				MessageDeduplicationId: mdl.Box("bar"),
				Body:                   mdl.Box(`{"attributes":{"encoding":"application/json","sqsDelaySeconds":"45","sqsMessageDeduplicationId":"bar","sqsMessageGroupId":"foo"},"body":"{\"Foo\":\"bar\"}"}`),
			},
		},
	}

	for test, data := range tests {
		t.Run(test, func(t *testing.T) {
			logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))

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

	largeBody := funk.Repeat(BodyStruct{Foo: "bar"}, 1000)
	largeAttributes := funk.Repeat(
		map[string]string{sqs.AttributeSqsMessageGroupId: "foo"},
		1000)
	largeExpectedSqsMessage := funk.Repeat(&sqs.Message{
		MessageGroupId: mdl.Box("foo"),
		Body:           mdl.Box(`{"attributes":{"encoding":"application/json","sqsMessageGroupId":"foo"},"body":"{\"Foo\":\"bar\"}"}`),
	}, 1000)

	tests := map[string]struct {
		attributes         []map[string]string
		body               []BodyStruct
		expectedSqsMessage []*sqs.Message
	}{
		"single": {
			attributes: []map[string]string{
				{},
			},
			body: []BodyStruct{
				{Foo: "bar"},
			},
			expectedSqsMessage: []*sqs.Message{
				{
					Body: mdl.Box(`{"attributes":{"encoding":"application/json"},"body":"{\"Foo\":\"bar\"}"}`),
				},
			},
		},
		"multiple": {
			attributes: []map[string]string{
				{sqs.AttributeSqsDelaySeconds: "45"},
				{sqs.AttributeSqsMessageGroupId: "foo"},
				{sqs.AttributeSqsMessageGroupId: "foo1"},
				{sqs.AttributeSqsMessageGroupId: "foo2"},
				{sqs.AttributeSqsMessageGroupId: "foo3"},
				{sqs.AttributeSqsMessageGroupId: "foo4"},
				{sqs.AttributeSqsMessageGroupId: "bar"},
				{sqs.AttributeSqsMessageDeduplicationId: "bar"},
				{
					sqs.AttributeSqsDelaySeconds:           "45",
					sqs.AttributeSqsMessageGroupId:         "foo1",
					sqs.AttributeSqsMessageDeduplicationId: "bar1",
				},
				{
					sqs.AttributeSqsDelaySeconds:           "46",
					sqs.AttributeSqsMessageGroupId:         "foo2",
					sqs.AttributeSqsMessageDeduplicationId: "bar2",
				},
				{
					sqs.AttributeSqsDelaySeconds:           "47",
					sqs.AttributeSqsMessageGroupId:         "foo3",
					sqs.AttributeSqsMessageDeduplicationId: "bar3",
				},
				{
					sqs.AttributeSqsDelaySeconds:           "48",
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
					Body:         mdl.Box(`{"attributes":{"encoding":"application/json","sqsDelaySeconds":"45"},"body":"{\"Foo\":\"bar\"}"}`),
				},
				{
					MessageGroupId: mdl.Box("foo"),
					Body:           mdl.Box(`{"attributes":{"encoding":"application/json","sqsMessageGroupId":"foo"},"body":"{\"Foo\":\"bar\"}"}`),
				},
				{
					MessageGroupId: mdl.Box("foo1"),
					Body:           mdl.Box(`{"attributes":{"encoding":"application/json","sqsMessageGroupId":"foo1"},"body":"{\"Foo\":\"bar1\"}"}`),
				},
				{
					MessageGroupId: mdl.Box("foo2"),
					Body:           mdl.Box(`{"attributes":{"encoding":"application/json","sqsMessageGroupId":"foo2"},"body":"{\"Foo\":\"bar2\"}"}`),
				},
				{
					MessageGroupId: mdl.Box("foo3"),
					Body:           mdl.Box(`{"attributes":{"encoding":"application/json","sqsMessageGroupId":"foo3"},"body":"{\"Foo\":\"bar3\"}"}`),
				},
				{
					MessageGroupId: mdl.Box("foo4"),
					Body:           mdl.Box(`{"attributes":{"encoding":"application/json","sqsMessageGroupId":"foo4"},"body":"{\"Foo\":\"bar4\"}"}`),
				},
				{
					MessageGroupId: mdl.Box("bar"),
					Body:           mdl.Box(`{"attributes":{"encoding":"application/json","sqsMessageGroupId":"bar"},"body":"{\"Foo\":\"bar\"}"}`),
				},
				{
					MessageDeduplicationId: mdl.Box("bar"),
					Body:                   mdl.Box(`{"attributes":{"encoding":"application/json","sqsMessageDeduplicationId":"bar"},"body":"{\"Foo\":\"bar\"}"}`),
				},
				{
					DelaySeconds:           45,
					MessageGroupId:         mdl.Box("foo1"),
					MessageDeduplicationId: mdl.Box("bar1"),
					Body:                   mdl.Box(`{"attributes":{"encoding":"application/json","sqsDelaySeconds":"45","sqsMessageDeduplicationId":"bar1","sqsMessageGroupId":"foo1"},"body":"{\"Foo\":\"multipleAttributes1\"}"}`),
				},
				{
					DelaySeconds:           46,
					MessageGroupId:         mdl.Box("foo2"),
					MessageDeduplicationId: mdl.Box("bar2"),
					Body:                   mdl.Box(`{"attributes":{"encoding":"application/json","sqsDelaySeconds":"46","sqsMessageDeduplicationId":"bar2","sqsMessageGroupId":"foo2"},"body":"{\"Foo\":\"multipleAttributes2\"}"}`),
				},
				{
					DelaySeconds:           47,
					MessageGroupId:         mdl.Box("foo3"),
					MessageDeduplicationId: mdl.Box("bar3"),
					Body:                   mdl.Box(`{"attributes":{"encoding":"application/json","sqsDelaySeconds":"47","sqsMessageDeduplicationId":"bar3","sqsMessageGroupId":"foo3"},"body":"{\"Foo\":\"multipleAttributes3\"}"}`),
				},
				{
					DelaySeconds:           48,
					MessageGroupId:         mdl.Box("foo4"),
					MessageDeduplicationId: mdl.Box("bar4"),
					Body:                   mdl.Box(`{"attributes":{"encoding":"application/json","sqsDelaySeconds":"48","sqsMessageDeduplicationId":"bar4","sqsMessageGroupId":"foo4"},"body":"{\"Foo\":\"multipleAttributes4\"}"}`),
				},
			},
		},
		"large": {
			attributes:         largeAttributes,
			body:               largeBody,
			expectedSqsMessage: largeExpectedSqsMessage,
		},
	}

	for test, data := range tests {
		t.Run(test, func(t *testing.T) {
			logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))

			expectedSqsMessageChunks := funk.Chunk(data.expectedSqsMessage, stream.SqsOutputBatchSize)

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
