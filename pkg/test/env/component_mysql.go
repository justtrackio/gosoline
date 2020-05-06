package env

import (
	"github.com/Masterminds/squirrel"
	"github.com/applike/gosoline/pkg/application"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

type mysqlComponent struct {
	baseComponent
	client      *sqlx.DB
	credentials mysqlCredentials
	binding     containerBinding
}

func (c *mysqlComponent) AppOptions() []application.Option {
	return []application.Option{
		application.WithConfigMap(map[string]interface{}{
			"db_hostname":     c.binding.host,
			"db_username":     c.credentials.UserName,
			"db_password":     c.credentials.UserPassword,
			"db_database":     c.credentials.DatabaseName,
			"db_port":         c.binding.port,
			"db_auto_migrate": true,
		}),
	}
}

func (c *mysqlComponent) Client() *sqlx.DB {
	return c.client
}

func (c *mysqlComponent) Exec(qry string, args ...interface{}) {
	_, err := c.client.Exec(qry, args...)

	if err != nil {
		assert.FailNow(c.t, err.Error(), "failed to execute query")
		return
	}
}

func (c *mysqlComponent) AssertRowCount(table string, expectedCount int) {
	qry, args, err := squirrel.Select("COUNT(*)").From(table).ToSql()

	if err != nil {
		assert.FailNow(c.t, err.Error(), "can not generate qry to count rows in table %s", table)
	}

	var actualCount int
	err = c.client.Get(&actualCount, qry, args...)

	if err != nil {
		assert.FailNow(c.t, err.Error(), "can not count rows in table %s", table)
	}

	assert.Equal(c.t, expectedCount, actualCount, "row count doesn't match for table %s", table)
}
