package ddb

import (
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/hashicorp/go-multierror"
)

const (
	ErrorConditionalCheckFailed            = errorConditionalCheckFailed("conditional check failed")
	ErrorTransactionConflict               = errorTransactionConflict("transaction conflict")
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

func parseTransactionError(err error) error {
	multiErr := &multierror.Error{}

	var tcErr *types.TransactionCanceledException
	if !errors.As(err, &tcErr) {
		return err
	}

	for _, r := range tcErr.CancellationReasons {
		if *r.Code == cancellationReasonNone {
			continue
		}

		switch *r.Code {
		case cancellationReasonConditionCheckFailed:
			multiErr = multierror.Append(multiErr, ErrorConditionalCheckFailed)
		case cancellationReasonTransactionConflict:
			multiErr = multierror.Append(multiErr, ErrorTransactionConflict)
		default:
			multiErr = multierror.Append(multiErr, errors.New(*r.Code))
		}
	}

	return multiErr.ErrorOrNil()
}
