//go:build integration && fixtures

package integration

import (
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type testSuite struct {
	suite.Suite

	clock clock.Clock
}

func Test_RunTestSuite(t *testing.T) {
	suite.Run(t, &testSuite{})
}

func (s *testSuite) SetupSuite() []suite.Option {
	s.clock = clock.NewFakeClockAt(time.Date(2022, 1, 2, 3, 4, 5, 0, time.UTC))

	return []suite.Option{
		suite.WithLogLevel("info"),
		suite.WithConfigFile("./config.dist.yml"),
		suite.WithConfigFile("./config.test.yml"),
		suite.WithModule("app", newAppModule),
		suite.WithClockProvider(s.clock),
	}
}

func (s *testSuite) Test_DynamoDB(app suite.AppUnderTest) {
	app.WaitDone()

	repository, err := s.Env().DynamoDb("default").Repository(ddbSettings)
	if err != nil {
		s.FailNow("unable to initialize repository: %w", err)
	}

	qb := repository.GetItemBuilder().WithHash(uint(1))

	it := item{}
	_, err = repository.GetItem(s.Env().Context(), qb, &it)
	if err != nil {
		s.FailNow("unable to fetch item: %w", err)
	}

	expected := item{
		Id:        1,
		CreatedAt: s.clock.Now(),
		UpdatedAt: s.clock.Now(),
	}

	s.Equal(expected, it)
}

func (s *testSuite) Test_MySql(app suite.AppUnderTest) {
	app.WaitDone()

	sql := s.Env().MySql("default").Client()

	row := sql.QueryRow("select * from items where id = ?", 1)

	var id uint
	var createdAt, updatedAt time.Time

	err := row.Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		s.FailNow("unable to scan row: %w", err)
	}

	it := item{
		Id:        id,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	expected := item{
		Id:        1,
		CreatedAt: s.clock.Now(),
		UpdatedAt: s.clock.Now(),
	}

	s.Equal(expected, it)
}
