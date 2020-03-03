//+build integration

package test_test

import (
	pkgTest "github.com/applike/gosoline/pkg/test"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_mysql(t *testing.T) {
	setup(t)

	mocks := pkgTest.Boot("test_configs/config.mysql.test.yml")
	defer mocks.Shutdown()

	db := mocks.ProvideMysqlClient("mysql")

	err := db.Ping()
	assert.NoError(t, err)
}
