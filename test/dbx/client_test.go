//go:build integration

package mysql_test

import (
	"context"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/dbx"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type ToDo struct {
	Id        int       `db:"id"`
	Name      string    `db:"name"`
	Index     int       `db:"index"`
	CreatedAt time.Time `db:"created_at"`
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

type ClientTestSuite struct {
	suite.Suite
	client dbx.Client[ToDo]
}

func (s *ClientTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithConfigFile("config.yml"),
		suite.WithClockProvider(clock.NewRealClock()),
	}
}

func (s *ClientTestSuite) SetupTest() (err error) {
	s.client, err = dbx.NewClient[ToDo](s.Env().Context(), s.Env().Config(), s.Env().Logger(), "default", "todo")

	return
}

func (s *ClientTestSuite) TestStatements() {
	ctx := context.Background()
	todo := ToDo{
		Name:      "foobar",
		Index:     1,
		CreatedAt: time.Now(),
	}

	res, err := s.client.Insert(todo).Exec(ctx)
	s.NoError(err, "there should not be an error on insert")

	id, err := res.LastInsertId()
	s.NoError(err, "there should not be an error on getting the last insert id")
	s.Equal(int64(1), id)

	affectedRows, err := res.RowsAffected()
	s.NoError(err, "there should not be an error on getting the affected rows")
	s.Equal(int64(1), affectedRows)

	todos, err := s.client.Select().Exec(ctx)
	s.NoError(err, "there should not be an error on select")
	s.Len(todos, 1)
	s.Equal("foobar", todos[0].Name, "the name of the todo should be foobar")

	res, err = s.client.Update().Set("name", "foobaz").Where(dbx.Eq{"id": 1}).Exec(ctx)
	s.NoError(err, "there should not be an error on update")

	affectedRows, err = res.RowsAffected()
	s.NoError(err, "there should not be an error on getting the affected rows")
	s.Equal(int64(1), affectedRows)

	todos, err = s.client.Select().Exec(ctx)
	s.NoError(err, "there should not be an error on select")
	s.Len(todos, 1)
	s.Equal("foobaz", todos[0].Name, "the name of the todo should be foobaz")

	todo = ToDo{
		Id:        1,
		Name:      "foo replace",
		Index:     2,
		CreatedAt: time.Now(),
	}
	res, err = s.client.Replace(todo).Exec(ctx)
	s.NoError(err, "there should not be an error on replace")

	id, err = res.LastInsertId()
	s.NoError(err, "there should not be an error on getting the last insert id")
	s.Equal(int64(1), id)

	affectedRows, err = res.RowsAffected()
	s.NoError(err, "there should not be an error on getting the affected rows")
	s.Equal(int64(2), affectedRows)

	todos, err = s.client.Select().Exec(ctx)
	s.NoError(err, "there should not be an error on select")
	s.Len(todos, 1)
	s.Equal("foo replace", todos[0].Name, "the name of the todo should be foo replace")

	res, err = s.client.Delete().Where(dbx.Eq{"id": 1}).Exec(ctx)
	s.NoError(err, "there should not be an error on delete")

	affectedRows, err = res.RowsAffected()
	s.NoError(err, "there should not be an error on getting the affected rows")
	s.Equal(int64(1), affectedRows)

	todos, err = s.client.Select().Exec(ctx)
	s.NoError(err, "there should not be an error on select")
	s.Len(todos, 0, "there should be no todos left after deletion")
}
