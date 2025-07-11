//go:build integration

package mysql_test

import (
	"context"
	"testing"
	"time"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
	"github.com/jmoiron/sqlx"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type ToDo struct {
	Id        uint                         `db:"id"`
	Name      string                       `db:"name"`
	Data      db.JSON[string, db.Nullable] `db:"data"`
	CreatedAd *time.Time                   `db:"created_at"`
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

type ClientTestSuite struct {
	suite.Suite
}

func (s *ClientTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithConfigFile("config.yml"),
		suite.WithClockProvider(clock.NewRealClock()),
	}
}

func (s *ClientTestSuite) TestConnectionRefused() {
	var client db.Client
	var proxy *toxiproxy.Proxy
	failedAttempts := 0

	client, proxy = s.getClients("default", func(err error, duration time.Duration) {
		failedAttempts++

		if failedAttempts == 3 {
			perr := proxy.Enable()
			s.FailIfError(perr)
		}
	})
	err := proxy.Disable()
	s.FailIfError(err)

	_, err = client.Exec(context.Background(), "SELECT * FROM todo")
	s.NoError(err)

	s.Equal(3, failedAttempts)
}

func (s *ClientTestSuite) TestReadIOTimeout() {
	var client db.Client
	var proxy *toxiproxy.Proxy
	failedAttempts := 0

	client, proxy = s.getClients("default", func(err error, duration time.Duration) {
		failedAttempts++

		if failedAttempts == 3 {
			perr := proxy.RemoveToxic("latency_down")
			s.FailIfError(perr)
		}
	})

	_, err := proxy.AddToxic("latency_down", "latency", "downstream", 1.0, toxiproxy.Attributes{
		"latency": 200,
	})
	s.FailIfError(err)

	rows, err := client.Queryx(context.Background(), "SELECT * FROM todo")
	s.FailIfError(err)

	rowsCount := 0
	for rows.Next() {
		rowsCount++
	}

	s.Equal(1, rowsCount)
}

func (s *ClientTestSuite) TestRead() {
	client, _ := s.getClients("default", func(err error, duration time.Duration) {})

	todo := &ToDo{}
	err := client.Get(context.Background(), todo, "SELECT * FROM todo")
	s.FailIfError(err)
}

func (s *ClientTestSuite) getClients(name string, notifier exec.Notify) (db.Client, *toxiproxy.Proxy) {
	var err error
	var connection *sqlx.DB

	ctx := s.Env().Context()
	config := s.Env().Config()
	logger := s.Env().Logger()

	proxy := s.Env().MySql(name).Toxiproxy()
	connection, err = db.NewConnection(ctx, config, logger, name)
	s.FailIfError(err)

	executor, err := db.NewExecutor(config, logger, name, "api", notifier)
	s.FailIfError(err)
	client := db.NewClientWithInterfaces(logger, connection, executor)

	return client, proxy
}
