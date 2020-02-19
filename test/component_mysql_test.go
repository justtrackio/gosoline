//+build integration

package test_test

import (
	"fmt"
	pkgTest "github.com/applike/gosoline/pkg/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"net/url"
	"testing"
)

func Test_mysql(t *testing.T) {
	setup(t)

	pkgTest.Boot("test_configs/config.mysql.test.yml")
	defer pkgTest.Shutdown()

	dsn := url.URL{
		User: url.UserPassword("root", "gosoline"),
		Host: fmt.Sprintf("tcp(%s:%v)", "172.17.0.1", "3333"),
		Path: "myDbName",
	}

	qry := dsn.Query()
	dsn.RawQuery = qry.Encode()

	db, err := sqlx.Open("mysql", dsn.String()[2:])
	assert.NoError(t, err)

	err = db.Ping()
	assert.NoError(t, err)
}
