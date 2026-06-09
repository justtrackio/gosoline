package httpserver_test

import (
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	httpserverMocks "github.com/justtrackio/gosoline/pkg/httpserver/mocks"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/testify/suite"
)

func TestRunConnectionPressureManagerTestSuite(t *testing.T) {
	suite.Run(t, new(ConnectionPressureManagerTestSuite))
}

type ConnectionPressureManagerTestSuite struct {
	suite.Suite

	recorder *httpserverMocks.ServerMetricRecorder
	manager  httpserver.ConnectionPressureManager
}

func (s *ConnectionPressureManagerTestSuite) SetupTest() {
	s.recorder = httpserverMocks.NewServerMetricRecorder(s.T())
	s.manager = httpserver.NewConnectionPressureManagerWithInterfaces(s.T().Context(), clock.NewFakeClock(), s.recorder)
}

func (s *ConnectionPressureManagerTestSuite) TestTracksConnectionOpenedAndClosed() {
	conn := httpserverMocks.NewNetConn(s.T())
	s.recorder.EXPECT().TrackConnectionOpened(matcher.Context).Return().Once()
	s.recorder.EXPECT().TrackConnectionClosed(matcher.Context).Return().Once()

	s.manager.ConnState(conn, http.StateNew)
	s.manager.ConnState(conn, http.StateActive)
	s.manager.ConnState(conn, http.StateIdle)
	s.manager.ConnState(conn, http.StateClosed)
}

func (s *ConnectionPressureManagerTestSuite) TestNotifiesWhenConnectionBecomesIdle() {
	conn := httpserverMocks.NewNetConn(s.T())
	ready := make(chan struct{})
	notified := atomic.Bool{}
	s.recorder.EXPECT().TrackConnectionOpened(matcher.Context).Return().Once()

	go func() {
		close(ready)
		<-s.manager.IdleConnectionAvailable()
		notified.Store(true)
	}()
	<-ready

	s.manager.ConnState(conn, http.StateNew)
	s.manager.ConnState(conn, http.StateIdle)

	s.Eventually(notified.Load, time.Second, time.Millisecond)
}

func (s *ConnectionPressureManagerTestSuite) TestClosesOldestIdleConnection() {
	connA := httpserverMocks.NewNetConn(s.T())
	connB := httpserverMocks.NewNetConn(s.T())
	connAClosed := atomic.Bool{}
	connA.EXPECT().Close().Run(func() {
		connAClosed.Store(true)
	}).Return(nil).Once()
	s.recorder.EXPECT().TrackConnectionOpened(matcher.Context).Return().Twice()
	s.recorder.EXPECT().TrackConnectionClosed(matcher.Context).Return().Once()

	s.manager.ConnState(connA, http.StateNew)
	s.manager.ConnState(connA, http.StateIdle)
	s.manager.ConnState(connB, http.StateNew)
	s.manager.ConnState(connA, http.StateIdle)

	closed, err := s.manager.CloseOldestIdleConnection()

	s.NoError(err)
	s.True(closed)
	s.True(connAClosed.Load())
}

func (s *ConnectionPressureManagerTestSuite) TestDoesNotCloseActiveConnection() {
	conn := httpserverMocks.NewNetConn(s.T())
	s.recorder.EXPECT().TrackConnectionOpened(matcher.Context).Return().Once()

	s.manager.ConnState(conn, http.StateNew)

	closed, err := s.manager.CloseOldestIdleConnection()

	s.NoError(err)
	s.False(closed)
}

func (s *ConnectionPressureManagerTestSuite) TestTracksClosedOnceWhenIdleConnectionWasAlreadyEvicted() {
	conn := httpserverMocks.NewNetConn(s.T())
	conn.EXPECT().Close().Return(nil).Once()
	s.recorder.EXPECT().TrackConnectionOpened(matcher.Context).Return().Once()
	s.recorder.EXPECT().TrackConnectionClosed(matcher.Context).Return().Once()

	s.manager.ConnState(conn, http.StateNew)
	s.manager.ConnState(conn, http.StateIdle)

	closed, err := s.manager.CloseOldestIdleConnection()
	s.NoError(err)
	s.True(closed)

	s.manager.ConnState(conn, http.StateClosed)
}
