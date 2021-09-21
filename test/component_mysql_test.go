//go:build integration
// +build integration

package test_test

import (
	"testing"

	pkgTest "github.com/justtrackio/gosoline/pkg/test"
	"github.com/stretchr/testify/assert"
)

func Test_mysql(t *testing.T) {
	t.Parallel()
	setup(t)

	mocks, err := pkgTest.Boot("test_configs/config.mysql.test.yml")
	defer func() {
		if mocks != nil {
			mocks.Shutdown()
		}
	}()

	if err != nil {
		assert.Fail(t, "failed to boot mocks: %s", err.Error())

		return
	}

	db := mocks.ProvideMysqlClient("mysql")

	err = db.Ping()
	assert.NoError(t, err)
}
