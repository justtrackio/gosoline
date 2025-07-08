package httpserver_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/stretchr/testify/suite"
)

func TestRunTrafficDistributorTestSuite(t *testing.T) {
	suite.Run(t, new(TrafficDistributorTestSuite))
}

type TrafficDistributorTestSuite struct {
	suite.Suite
	clock clock.FakeClock

	distributor httpserver.TrafficDistributor
}

func (s *TrafficDistributorTestSuite) SetupTest() {
	s.clock = clock.NewFakeClock()
	s.distributor = httpserver.NewTrafficDistributorWithInterfaces(s.clock, httpserver.TrafficDistributorSettings{
		Enabled:                   true,
		MaxConnectionAge:          time.Second,
		MaxConnectionRequestCount: 2,
	})
}

func (s *TrafficDistributorTestSuite) TestShouldCloseConnection_Age() {
	remoteAddr := "127.0.0.1:12345"
	headers := http.Header{}

	s.False(s.distributor.ShouldCloseConnection(remoteAddr, headers))

	s.clock.Advance(time.Second + time.Millisecond)

	s.True(s.distributor.ShouldCloseConnection(remoteAddr, headers))
	s.False(s.distributor.ShouldCloseConnection(remoteAddr, headers))
}

func (s *TrafficDistributorTestSuite) TestShouldCloseConnection_RequestCount() {
	remoteAddr := "127.0.0.1:12345"
	headers := http.Header{}

	s.False(s.distributor.ShouldCloseConnection(remoteAddr, headers))
	s.False(s.distributor.ShouldCloseConnection(remoteAddr, headers))
	s.True(s.distributor.ShouldCloseConnection(remoteAddr, headers))
	s.False(s.distributor.ShouldCloseConnection(remoteAddr, headers))
}

func (s *TrafficDistributorTestSuite) TestShouldCloseConnection_AgeOnly() {
	distributor := httpserver.NewTrafficDistributorWithInterfaces(s.clock, httpserver.TrafficDistributorSettings{
		Enabled:                   true,
		MaxConnectionAge:          time.Second,
		MaxConnectionRequestCount: 0,
	})

	remoteAddr := "127.0.0.1:12345"
	headers := http.Header{}

	s.False(distributor.ShouldCloseConnection(remoteAddr, headers))
	s.False(distributor.ShouldCloseConnection(remoteAddr, headers))
	s.False(distributor.ShouldCloseConnection(remoteAddr, headers))

	s.clock.Advance(time.Second + time.Millisecond)

	s.True(distributor.ShouldCloseConnection(remoteAddr, headers))
	s.False(distributor.ShouldCloseConnection(remoteAddr, headers))
}

func (s *TrafficDistributorTestSuite) TestShouldCloseConnection_RequestCountOnly() {
	distributor := httpserver.NewTrafficDistributorWithInterfaces(s.clock, httpserver.TrafficDistributorSettings{
		Enabled:                   true,
		MaxConnectionAge:          0,
		MaxConnectionRequestCount: 2,
	})

	remoteAddr := "127.0.0.1:12345"
	headers := http.Header{}

	s.False(distributor.ShouldCloseConnection(remoteAddr, headers))

	s.clock.Advance(time.Second + time.Millisecond)

	s.False(distributor.ShouldCloseConnection(remoteAddr, headers))
	s.True(distributor.ShouldCloseConnection(remoteAddr, headers))
	s.False(distributor.ShouldCloseConnection(remoteAddr, headers))
}

func (s *TrafficDistributorTestSuite) TestShouldCloseConnection_MultipleHosts() {
	remoteAddrA := "127.0.0.1:12345"
	remoteAddrB := "127.0.0.1:54321"
	headers := http.Header{}

	// First host almost at limit
	s.False(s.distributor.ShouldCloseConnection(remoteAddrA, headers))
	s.False(s.distributor.ShouldCloseConnection(remoteAddrA, headers))

	// Second host is not affected
	s.False(s.distributor.ShouldCloseConnection(remoteAddrB, headers))

	// First host is now closed
	s.True(s.distributor.ShouldCloseConnection(remoteAddrA, headers))

	// Second host is still not affected and then gets closed
	s.False(s.distributor.ShouldCloseConnection(remoteAddrB, headers))
	s.True(s.distributor.ShouldCloseConnection(remoteAddrB, headers))

	// First host can connect again
	s.False(s.distributor.ShouldCloseConnection(remoteAddrA, headers))

	// Second host can connect again
	s.False(s.distributor.ShouldCloseConnection(remoteAddrB, headers))
}
