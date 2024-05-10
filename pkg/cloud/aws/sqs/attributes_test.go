package sqs_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cloud/aws/sqs"
	"github.com/stretchr/testify/assert"
)

func TestAttributeEncodeHandler_Encode(t *testing.T) {
	testCases := map[string]struct {
		attr               sqs.Attribute
		provider           sqs.AttributeProvider
		msg                any
		expectedAttributes map[string]any
	}{
		"delay_seconds": {
			attr: sqs.MessageDelaySeconds,
			provider: func(data any) (any, error) {
				return data, nil
			},
			msg: int32(5),
			expectedAttributes: map[string]any{
				sqs.AttributeSqsDelaySeconds: int32(5),
			},
		},
		"group_id": {
			attr: sqs.MessageGroupId,
			provider: func(data any) (any, error) {
				return data, nil
			},
			msg: "foo",
			expectedAttributes: map[string]any{
				sqs.AttributeSqsMessageGroupId: "foo",
			},
		},
		"deduplication_id": {
			attr: sqs.MessageDeduplicationId,
			provider: func(data any) (any, error) {
				return data, nil
			},
			msg: "bar",
			expectedAttributes: map[string]any{
				sqs.AttributeSqsMessageDeduplicationId: "bar",
			},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			h := sqs.NewAttributeEncodeHandler(test.attr, test.provider)
			ctx, attributes, err := h.Encode(t.Context(), test.msg, make(map[string]any))
			assert.NoError(t, err)
			assert.Equal(t, t.Context(), ctx)
			assert.Equal(t, test.expectedAttributes, attributes)
		})
	}
}
