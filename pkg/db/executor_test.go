package db_test

import (
	"io"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/stretchr/testify/assert"
)

func TestCheckMySqlErrors(t *testing.T) {
	testCases := []struct {
		err *mysql.MySQLError
		res exec.ErrorType
	}{
		{
			err: buildMysqlError(9005, "HY000", "Region is unavailable"),
			res: exec.ErrorTypeRetryable,
		},
		{
			err: buildMysqlError(1792, "25006", "Cannot execute statement in a READ ONLY transaction."),
			res: exec.ErrorTypeUnknown,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.err.Error(), func(t *testing.T) {
			res := db.CheckRegionUnavailable(nil, tc.err)
			assert.Equal(t, tc.res, res)
		})
	}
}

func TestCheckIoUnexpectedEof(t *testing.T) {
	res := db.CheckIoUnexpectedEof(nil, io.ErrUnexpectedEOF)
	assert.Equal(t, exec.ErrorTypeRetryable, res)
}

func buildMysqlError(number uint16, state string, msg string) *mysql.MySQLError {
	err := &mysql.MySQLError{
		Number:   number,
		SQLState: [5]byte{},
		Message:  msg,
	}
	copy(err.SQLState[:], state)

	return err
}
