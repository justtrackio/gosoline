package sqs_test

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cloud/aws/sqs"
	"github.com/stretchr/testify/assert"
)

func TestAttributeEncodeHandler_Encode(t *testing.T) {
	testCases := map[string]struct {
		attr               sqs.Attribute
		provider           sqs.AttributeProvider
		msg                interface{}
		expectedAttributes map[string]interface{}
	}{
		"delay_seconds": {
			attr: sqs.MessageDelaySeconds,
			provider: func(data interface{}) (interface{}, error) {
				return data, nil
			},
			msg: int32(5),
			expectedAttributes: map[string]interface{}{
				sqs.AttributeSqsDelaySeconds: int32(5),
			},
		},
		"group_id": {
			attr: sqs.MessageGroupId,
			provider: func(data interface{}) (interface{}, error) {
				return data, nil
			},
			msg: "foo",
			expectedAttributes: map[string]interface{}{
				sqs.AttributeSqsMessageGroupId: "foo",
			},
		},
		"deduplication_id": {
			attr: sqs.MessageDeduplicationId,
			provider: func(data interface{}) (interface{}, error) {
				return data, nil
			},
			msg: "bar",
			expectedAttributes: map[string]interface{}{
				sqs.AttributeSqsMessageDeduplicationId: "bar",
			},
		},
	}

	for name, test := range testCases {
		test := test
		t.Run(name, func(t *testing.T) {
			h := sqs.NewAttributeEncodeHandler(test.attr, test.provider)
			ctx, attributes, err := h.Encode(context.Background(), test.msg, make(map[string]interface{}))
			assert.NoError(t, err)
			assert.Equal(t, context.Background(), ctx)
			assert.Equal(t, test.expectedAttributes, attributes)
		})
	}
}
