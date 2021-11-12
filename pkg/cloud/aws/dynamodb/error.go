package dynamodb

import (
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/hashicorp/go-multierror"
)

const (
	ConditionalCheckFailedError            = errorConditionalCheckFailed("conditional check failed")
	TransactionConflictError               = errorTransactionConflict("transaction conflict")
	cancellationReasonNone                 = "None"
	cancellationReasonConditionCheckFailed = "ConditionalCheckFailed"
	cancellationReasonTransactionConflict  = "TransactionConflict"
)

type errorConditionalCheckFailed string

func (e errorConditionalCheckFailed) Error() string {
	return string(e)
}

type errorTransactionConflict string

func (e errorTransactionConflict) Error() string {
	return string(e)
}

func TransformTransactionError(err error) error {
	var tcoErr *types.TransactionConflictException
	if errors.As(err, &tcoErr) {
		return TransactionConflictError
	}

	var tcaErr *types.TransactionCanceledException
	if !errors.As(err, &tcaErr) {
		return err
	}

	multiErr := &multierror.Error{}
	for _, r := range tcaErr.CancellationReasons {
		switch *r.Code {
		case cancellationReasonNone:
			continue
		case cancellationReasonTransactionConflict:
			multiErr = multierror.Append(multiErr, TransactionConflictError)
		case cancellationReasonConditionCheckFailed:
			multiErr = multierror.Append(multiErr, ConditionalCheckFailedError)
		default:
			multiErr = multierror.Append(multiErr, errors.New(*r.Code))
		}
	}

	return multiErr.ErrorOrNil()
}

type RetryOnTransactionConflict struct{}

// IsErrorRetryable reacts on TransactionConflicts within transactional operations
// Transaction Conflicts occur when using a transactions and multiple calls want to change the same resource.
// This is usually resolvable by retrying
func (r RetryOnTransactionConflict) IsErrorRetryable(err error) aws.Ternary {
	err = TransformTransactionError(err)
	if errors.Is(err, TransactionConflictError) {
		return aws.TrueTernary
	}

	return aws.UnknownTernary
}

type RetryOnConditionalCheckFailed struct{}

// IsErrorRetryable reacts on failed ConditionalChecks within transactional operations
// When a condition fails within one transaction, this usually indicates an operation which should not be executed.
// Thus, it should never be retried
func (r RetryOnConditionalCheckFailed) IsErrorRetryable(err error) aws.Ternary {
	err = TransformTransactionError(err)
	if errors.Is(err, ConditionalCheckFailedError) {
		return aws.FalseTernary
	}

	return aws.UnknownTernary
}
