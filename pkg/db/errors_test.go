package db_test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/db"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"testing"
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
