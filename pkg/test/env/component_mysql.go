package env

import (
	"github.com/Masterminds/squirrel"
	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
	"github.com/jmoiron/sqlx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/stretchr/testify/assert"
)

type mysqlComponent struct {
	baseComponent
	client      *sqlx.DB
	credentials mysqlCredentials
	binding     containerBinding
	toxiproxy   *toxiproxy.Proxy
}

func (c *mysqlComponent) CfgOptions() []cfg.Option {
	return []cfg.Option{
		cfg.WithConfigMap(map[string]interface{}{
			"db": map[string]interface{}{
				c.name: map[string]interface{}{
					"uri.host":           c.binding.host,
					"uri.user":           c.credentials.UserName,
					"uri.password":       c.credentials.UserPassword,
					"uri.database":       c.credentials.DatabaseName,
					"uri.port":           c.binding.port,
					"migrations.enabled": true,
				},
			},
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

func (c *mysqlComponent) Toxiproxy() *toxiproxy.Proxy {
	return c.toxiproxy
}
