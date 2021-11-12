package dynamodb_test

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/dynamodb"
	"github.com/stretchr/testify/assert"
)

func TestTransformTransactionError(t *testing.T) {
	err := &types.TransactionCanceledException{
		CancellationReasons: []types.CancellationReason{
			{
				Code: aws.String("ConditionalCheckFailed"),
				Item: map[string]types.AttributeValue{
					"id":  &types.AttributeValueMemberN{Value: "42"},
					"rev": &types.AttributeValueMemberS{Value: "foo"},
					"foo": &types.AttributeValueMemberS{Value: "foo"},
				},
			},
			{
				Code: aws.String("TransactionConflict"),
				Item: map[string]types.AttributeValue{
					"id":  &types.AttributeValueMemberN{Value: "42"},
					"rev": &types.AttributeValueMemberS{Value: "foo"},
					"foo": &types.AttributeValueMemberS{Value: "foo"},
				},
			},
		},
	}

	transformed := dynamodb.TransformTransactionError(err)

	assert.True(t, errors.Is(transformed, dynamodb.ConditionalCheckFailedError))
	assert.True(t, errors.Is(transformed, dynamodb.TransactionConflictError))
}

func TestRetryOnTransactionConflict(t *testing.T) {
	retryable := &dynamodb.RetryOnTransactionConflict{}

	tcErr := &types.TransactionCanceledException{
		CancellationReasons: []types.CancellationReason{
			{
				Code: aws.String("TransactionConflict"),
			},
		},
	}

	ccfErr := &types.TransactionCanceledException{
		CancellationReasons: []types.CancellationReason{
			{
				Code: aws.String("ConditionalCheckFailed"),
			},
		},
	}

	tc := map[string]struct {
		err    error
		result aws.Ternary
	}{
		"nil": {
			err:    nil,
			result: aws.UnknownTernary,
		},
		"plain": {
			err:    errors.New("im an error"),
			result: aws.UnknownTernary,
		},
		"transaction conflict": {
			err:    tcErr,
			result: aws.TrueTernary,
		},
		"conditional check failed": {
			err:    ccfErr,
			result: aws.UnknownTernary,
		},
	}

	for name, c := range tc {
		t.Run(name, func(t *testing.T) {
			res := retryable.IsErrorRetryable(c.err)
			assert.Equal(t, c.result, res)
		})
	}
}

func TestRetryOnConditionalCheckFailed(t *testing.T) {
	retryable := &dynamodb.RetryOnConditionalCheckFailed{}

	tcErr := &types.TransactionCanceledException{
		CancellationReasons: []types.CancellationReason{
			{
				Code: aws.String("TransactionConflict"),
			},
		},
	}

	ccfErr := &types.TransactionCanceledException{
		CancellationReasons: []types.CancellationReason{
			{
				Code: aws.String("ConditionalCheckFailed"),
			},
		},
	}

	tc := map[string]struct {
		err    error
		result aws.Ternary
	}{
		"nil": {
			err:    nil,
			result: aws.UnknownTernary,
		},
		"plain": {
			err:    errors.New("im an error"),
			result: aws.UnknownTernary,
		},
		"transaction conflict": {
			err:    tcErr,
			result: aws.UnknownTernary,
		},
		"conditional check failed": {
			err:    ccfErr,
			result: aws.FalseTernary,
		},
	}

	for name, c := range tc {
		t.Run(name, func(t *testing.T) {
			res := retryable.IsErrorRetryable(c.err)
			assert.Equal(t, c.result, res)
		})
	}
}
