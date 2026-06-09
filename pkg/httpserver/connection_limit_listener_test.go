package httpserver_test

import (
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/httpserver"
	httpserverMocks "github.com/justtrackio/gosoline/pkg/httpserver/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/suite"
)

type ConnectionLimitListenerTestSuite struct {
	suite.Suite

	listener      *httpserverMocks.NetListener
	manager       *httpserverMocks.ConnectionPressureManager
	logger        logMocks.LoggerMock
	idleAvailable chan struct{}
}

func TestRunConnectionLimitListenerTestSuite(t *testing.T) {
	suite.Run(t, new(ConnectionLimitListenerTestSuite))
}

func (s *ConnectionLimitListenerTestSuite) SetupTest() {
	s.listener = httpserverMocks.NewNetListener(s.T())
	s.manager = httpserverMocks.NewConnectionPressureManager(s.T())
	s.logger = logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))
	s.idleAvailable = make(chan struct{})
}

func (s *ConnectionLimitListenerTestSuite) TestDisabledReturnsOriginalListener() {
	limited := s.newLimited(0)

	s.Same(s.listener, limited)
}

func (s *ConnectionLimitListenerTestSuite) TestAcceptWaitsUntilConnectionSlotIsReleased() {
	s.manager.EXPECT().CloseOldestIdleConnection().Return(false, nil).Maybe()
	s.manager.EXPECT().IdleConnectionAvailable().Return(s.idleAvailable).Maybe()

	limited := s.newLimited(1)

	s.mockAcceptedConn()
	limitedConnA, err := limited.Accept()
	s.NoError(err)
	s.NotNil(limitedConnA)

	acceptedSecond := atomic.Bool{}
	s.listener.EXPECT().Accept().Return(httpserverMocks.NewNetConn(s.T()), nil).Once()
	go func() {
		connB, acceptErr := limited.Accept()
		s.NoError(acceptErr)
		s.NotNil(connB)
		acceptedSecond.Store(true)
	}()

	s.Never(acceptedSecond.Load, 20*time.Millisecond, time.Millisecond)

	s.NoError(limitedConnA.Close())
	s.Eventually(acceptedSecond.Load, time.Second, time.Millisecond)
}

func (s *ConnectionLimitListenerTestSuite) TestClosesIdleConnectionBeforeWaiting() {
	closedIdle := atomic.Bool{}
	s.manager.EXPECT().CloseOldestIdleConnection().Run(func() {
		closedIdle.Store(true)
	}).Return(true, nil).Once()
	s.manager.EXPECT().CloseOldestIdleConnection().Return(false, nil).Maybe()
	s.manager.EXPECT().IdleConnectionAvailable().Return(s.idleAvailable).Maybe()

	limited := s.newLimited(1)

	s.mockAcceptedConn()
	limitedConnA, err := limited.Accept()
	s.NoError(err)
	s.NotNil(limitedConnA)

	s.listener.EXPECT().Accept().Return(httpserverMocks.NewNetConn(s.T()), nil).Once()
	acceptedSecond := atomic.Bool{}
	go func() {
		connB, acceptErr := limited.Accept()
		s.NoError(acceptErr)
		s.NotNil(connB)
		acceptedSecond.Store(true)
	}()

	s.Eventually(closedIdle.Load, time.Second, time.Millisecond)
	s.Never(acceptedSecond.Load, 20*time.Millisecond, time.Millisecond)

	s.NoError(limitedConnA.Close())
	s.Eventually(acceptedSecond.Load, time.Second, time.Millisecond)
}

func (s *ConnectionLimitListenerTestSuite) TestWakesUpWhenIdleConnectionBecomesAvailable() {
	s.manager.EXPECT().CloseOldestIdleConnection().Return(false, nil).Maybe()
	s.manager.EXPECT().IdleConnectionAvailable().Return(s.idleAvailable).Maybe()

	limited := s.newLimited(1)

	s.mockAcceptedConn()
	limitedConnA, err := limited.Accept()
	s.NoError(err)
	s.NotNil(limitedConnA)

	s.listener.EXPECT().Accept().Return(httpserverMocks.NewNetConn(s.T()), nil).Once()
	acceptedSecond := atomic.Bool{}
	go func() {
		connB, acceptErr := limited.Accept()
		s.NoError(acceptErr)
		s.NotNil(connB)
		acceptedSecond.Store(true)
	}()

	s.Never(acceptedSecond.Load, 20*time.Millisecond, time.Millisecond)

	s.NoError(limitedConnA.Close())
	close(s.idleAvailable)
	s.Eventually(acceptedSecond.Load, time.Second, time.Millisecond)
}

func (s *ConnectionLimitListenerTestSuite) TestCloseUnblocksWaitingAccept() {
	s.manager.EXPECT().CloseOldestIdleConnection().Return(false, nil).Maybe()
	s.manager.EXPECT().IdleConnectionAvailable().Return(s.idleAvailable).Maybe()
	s.listener.EXPECT().Close().Return(nil).Once()

	limited := s.newLimited(1)

	s.listener.EXPECT().Accept().Return(httpserverMocks.NewNetConn(s.T()), nil).Once()
	connA, err := limited.Accept()
	s.NoError(err)
	s.NotNil(connA)

	acceptedSecond := make(chan error, 1)
	acceptReturned := atomic.Bool{}
	go func() {
		_, acceptErr := limited.Accept()
		acceptReturned.Store(true)
		acceptedSecond <- acceptErr
	}()

	s.Never(acceptReturned.Load, 20*time.Millisecond, time.Millisecond)

	s.NoError(limited.Close())
	s.Eventually(acceptReturned.Load, time.Second, time.Millisecond)
	s.ErrorIs(<-acceptedSecond, net.ErrClosed)
}

func (s *ConnectionLimitListenerTestSuite) newLimited(maxConnections int) net.Listener {
	return httpserver.NewConnectionLimitListener(
		s.T().Context(),
		s.logger,
		s.listener,
		httpserver.ConcurrencySettings{MaxConnections: maxConnections},
		s.manager,
	)
}

func (s *ConnectionLimitListenerTestSuite) mockAcceptedConn() {
	connA := httpserverMocks.NewNetConn(s.T())
	connA.EXPECT().Close().Return(nil).Once()
	s.listener.EXPECT().Accept().Return(connA, nil).Once()
}
