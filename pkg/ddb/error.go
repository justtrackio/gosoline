package ddb

import (
	"errors"
	"fmt"
)

func IsTableNotFoundError(err error) bool {
	return errors.As(err, &TableNotFoundError{})
}

type TableNotFoundError struct {
	TableName string
	err       error
}

func NewTableNotFoundError(tableName string, err error) TableNotFoundError {
	return TableNotFoundError{
		TableName: tableName,
		err:       err,
	}
}

func (t TableNotFoundError) Error() string {
	return fmt.Sprintf("ddb table %s not found: %s", t.TableName, t.err)
}

func (t TableNotFoundError) Unwrap() error {
	return t.err
}
