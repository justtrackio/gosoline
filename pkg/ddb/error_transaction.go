package ddb

import (
	"errors"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/aws/aws-sdk-go/service/dynamodb"
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

func checkTransactionConflict(_ interface{}, err error) exec.ErrorType {
	if isTransactionCanceledException(err, ErrorTransactionConflict) {
		return exec.ErrorTypeRetryable
	}

	return exec.ErrorTypeUnknown
}

func checkPreconditionFailed(_ interface{}, err error) exec.ErrorType {
	if isTransactionCanceledException(err, ErrorConditionalCheckFailed) {
		return exec.ErrorTypePermanent
	}

	return exec.ErrorTypeUnknown
}

func isTransactionCanceledException(err error, checkErr error) bool {
	err = parseTransactionError(err)

	return errors.Is(err, checkErr)
}

func parseTransactionError(err error) error {
	multiErr := &multierror.Error{}

	var tcErr *dynamodb.TransactionCanceledException
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
