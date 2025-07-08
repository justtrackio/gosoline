package httpserver_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/stretchr/testify/suite"
)

func TestRunConnectionLifeCycleAdvisorTestSuite(t *testing.T) {
	suite.Run(t, new(ConnectionLifeCycleAdvisorTestSuite))
}

type ConnectionLifeCycleAdvisorTestSuite struct {
	suite.Suite
	clock clock.FakeClock

	advisor httpserver.ConnectionLifeCycleAdvisor
}

func (s *ConnectionLifeCycleAdvisorTestSuite) SetupTest() {
	s.clock = clock.NewFakeClock()
	s.advisor = httpserver.NewConnectionLifeCycleAdvisorWithInterfaces(s.clock, httpserver.ConnectionLifeCycleAdvisorSettings{
		Enabled:                   true,
		MaxConnectionAge:          time.Second,
		MaxConnectionRequestCount: 3,
	})
}

func (s *ConnectionLifeCycleAdvisorTestSuite) TestShouldCloseConnection_NotEnabled() {
	advisor := httpserver.NewConnectionLifeCycleAdvisorWithInterfaces(s.clock, httpserver.ConnectionLifeCycleAdvisorSettings{
		Enabled:                   false,
		MaxConnectionAge:          time.Second,
		MaxConnectionRequestCount: 2,
	})

	remoteAddr := "127.0.0.1:12345"
	headers := http.Header{}

	// Should always return false when not enabled
	s.False(advisor.ShouldCloseConnection(remoteAddr, headers))
	s.False(advisor.ShouldCloseConnection(remoteAddr, headers))
	s.clock.Advance(time.Second * 10)
	s.False(advisor.ShouldCloseConnection(remoteAddr, headers))
}

func (s *ConnectionLifeCycleAdvisorTestSuite) TestShouldCloseConnection_Age() {
	remoteAddr := "127.0.0.1:12345"
	headers := http.Header{}

	s.False(s.advisor.ShouldCloseConnection(remoteAddr, headers))

	s.clock.Advance(time.Second + time.Millisecond)

	s.True(s.advisor.ShouldCloseConnection(remoteAddr, headers))
	s.False(s.advisor.ShouldCloseConnection(remoteAddr, headers))
}

func (s *ConnectionLifeCycleAdvisorTestSuite) TestShouldCloseConnection_RequestCount() {
	remoteAddr := "127.0.0.1:12345"
	headers := http.Header{}

	s.False(s.advisor.ShouldCloseConnection(remoteAddr, headers))
	s.False(s.advisor.ShouldCloseConnection(remoteAddr, headers))
	s.True(s.advisor.ShouldCloseConnection(remoteAddr, headers))
	s.False(s.advisor.ShouldCloseConnection(remoteAddr, headers))
}

func (s *ConnectionLifeCycleAdvisorTestSuite) TestShouldCloseConnection_AgeOnly() {
	advisor := httpserver.NewConnectionLifeCycleAdvisorWithInterfaces(s.clock, httpserver.ConnectionLifeCycleAdvisorSettings{
		Enabled:                   true,
		MaxConnectionAge:          time.Second,
		MaxConnectionRequestCount: 0,
	})

	remoteAddr := "127.0.0.1:12345"
	headers := http.Header{}

	s.False(advisor.ShouldCloseConnection(remoteAddr, headers))
	s.False(advisor.ShouldCloseConnection(remoteAddr, headers))
	s.False(advisor.ShouldCloseConnection(remoteAddr, headers))

	s.clock.Advance(time.Second + time.Millisecond)

	s.True(advisor.ShouldCloseConnection(remoteAddr, headers))
	s.False(advisor.ShouldCloseConnection(remoteAddr, headers))
}

func (s *ConnectionLifeCycleAdvisorTestSuite) TestShouldCloseConnection_RequestCountOnly() {
	advisor := httpserver.NewConnectionLifeCycleAdvisorWithInterfaces(s.clock, httpserver.ConnectionLifeCycleAdvisorSettings{
		Enabled:                   true,
		MaxConnectionAge:          0,
		MaxConnectionRequestCount: 3,
	})

	remoteAddr := "127.0.0.1:12345"
	headers := http.Header{}

	s.False(advisor.ShouldCloseConnection(remoteAddr, headers))

	s.clock.Advance(time.Second + time.Millisecond)

	s.False(advisor.ShouldCloseConnection(remoteAddr, headers))
	s.True(advisor.ShouldCloseConnection(remoteAddr, headers))
	s.False(advisor.ShouldCloseConnection(remoteAddr, headers))
}

func (s *ConnectionLifeCycleAdvisorTestSuite) TestShouldCloseConnection_MultipleHosts() {
	remoteAddrA := "127.0.0.1:12345"
	remoteAddrB := "127.0.0.1:54321"
	headers := http.Header{}

	// First host almost at limit
	s.False(s.advisor.ShouldCloseConnection(remoteAddrA, headers))
	s.False(s.advisor.ShouldCloseConnection(remoteAddrA, headers))

	// Second host is not affected
	s.False(s.advisor.ShouldCloseConnection(remoteAddrB, headers))

	// First host is now closed
	s.True(s.advisor.ShouldCloseConnection(remoteAddrA, headers))

	// Second host is still not affected and then gets closed
	s.False(s.advisor.ShouldCloseConnection(remoteAddrB, headers))
	s.True(s.advisor.ShouldCloseConnection(remoteAddrB, headers))

	// First host can connect again
	s.False(s.advisor.ShouldCloseConnection(remoteAddrA, headers))

	// Second host can connect again
	s.False(s.advisor.ShouldCloseConnection(remoteAddrB, headers))
}
