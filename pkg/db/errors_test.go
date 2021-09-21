package db_test

import (
	"fmt"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/stretchr/testify/assert"
)

func TestIsDuplicateEntryError(t *testing.T) {
	valid := []error{
		&mysql.MySQLError{
			Number: 1062,
		},
		&db.DuplicateEntryError{},
		&db.DuplicateEntryError{
			Err: fmt.Errorf("some other error"),
		},
		fmt.Errorf("error: %w", &mysql.MySQLError{
			Number: 1062,
		}),
		fmt.Errorf("error: %w", &db.DuplicateEntryError{}),
	}

	invalid := []error{
		nil,
		fmt.Errorf("foo"),
		&mysql.MySQLError{
			Number: 42,
		},
	}

	for _, validErr := range valid {
		assert.True(t, db.IsDuplicateEntryError(validErr))
	}

	for _, invalidErr := range invalid {
		assert.False(t, db.IsDuplicateEntryError(invalidErr))
	}
}
