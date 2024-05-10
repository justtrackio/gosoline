package dbtx

import (
	"errors"
	"fmt"
)

type RecordNotFoundError struct {
	id      uint
	modelId string
	err     error
}

func NewRecordNotFoundError(id uint, modelId string, err error) RecordNotFoundError {
	return RecordNotFoundError{
		id:      id,
		modelId: modelId,
		err:     err,
	}
}

func (e RecordNotFoundError) Error() string {
	return fmt.Sprintf("could not find model of type %s with id %d: %s", e.modelId, e.id, e.err)
}

func (e RecordNotFoundError) Unwrap() error {
	return e.err
}

func IsRecordNotFoundError(err error) bool {
	return errors.As(err, &RecordNotFoundError{})
}

type NoQueryResultsError struct {
	modelId string
	err     error
}

func NewNoQueryResultsError(modelId string, err error) NoQueryResultsError {
	return NoQueryResultsError{
		modelId: modelId,
		err:     err,
	}
}

func (e NoQueryResultsError) Error() string {
	return fmt.Sprintf("could not find any results for model type %s: %s", e.modelId, e.err)
}

func (e NoQueryResultsError) Unwrap() error {
	return e.err
}

func IsNoQueryResultsError(err error) bool {
	return errors.As(err, &NoQueryResultsError{})
}

type TooManyResultsError struct {
	modelId  string
	found    int
	expected int
}

func NewTooManyResultsError(found int, expected int, modelId string) TooManyResultsError {
	return TooManyResultsError{
		modelId:  modelId,
		found:    found,
		expected: expected,
	}
}

func (e TooManyResultsError) Error() string {
	return fmt.Sprintf("found %d result(s), expected %d result(s) for model type %s", e.found, e.expected, e.modelId)
}

func IsTooManyResultsError(err error) bool {
	return errors.As(err, &TooManyResultsError{})
}
