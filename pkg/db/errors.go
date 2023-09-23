package db

import (
	"errors"
	"fmt"

	"github.com/VividCortex/mysqlerr"
	"github.com/go-sql-driver/mysql"
)

type DuplicateEntryError struct {
	Err error
}

func (e *DuplicateEntryError) Error() string {
	return fmt.Sprintf("duplicate entry: %s", e.Err.Error())
}

func (e *DuplicateEntryError) Is(err error) bool {
	_, ok := err.(*DuplicateEntryError)

	return ok
}

func (e *DuplicateEntryError) As(target interface{}) bool {
	targetErr, ok := target.(*DuplicateEntryError)

	if ok {
		*targetErr = *e
	}

	return ok
}

func (e *DuplicateEntryError) Unwrap() error {
	return e.Err
}

func IsDuplicateEntryError(err error) bool {
	mysqlErr := &mysql.MySQLError{}

	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == mysqlerr.ER_DUP_ENTRY
	}

	return errors.Is(err, &DuplicateEntryError{})
}
