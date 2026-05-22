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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectionPressureManager_TracksConnectionOpenedAndClosed(t *testing.T) {
	recorder := httpserverMocks.NewServerMetricRecorder(t)
	manager := httpserver.NewConnectionPressureManagerWithInterfaces(t.Context(), clock.NewFakeClock(), recorder)
	conn := httpserverMocks.NewNetConn(t)
	recorder.EXPECT().TrackConnectionOpened(matcher.Context).Return().Once()
	recorder.EXPECT().TrackConnectionClosed(matcher.Context).Return().Once()

	manager.ConnState(conn, http.StateNew)
	manager.ConnState(conn, http.StateActive)
	manager.ConnState(conn, http.StateIdle)
	manager.ConnState(conn, http.StateClosed)
}

func TestConnectionPressureManager_NotifiesWhenConnectionBecomesIdle(t *testing.T) {
	recorder := httpserverMocks.NewServerMetricRecorder(t)
	manager := httpserver.NewConnectionPressureManagerWithInterfaces(t.Context(), clock.NewFakeClock(), recorder)
	conn := httpserverMocks.NewNetConn(t)
	ready := make(chan struct{})
	notified := atomic.Bool{}
	recorder.EXPECT().TrackConnectionOpened(matcher.Context).Return().Once()
	go func() {
		close(ready)
		<-manager.IdleConnectionAvailable()
		notified.Store(true)
	}()
	<-ready

	manager.ConnState(conn, http.StateNew)
	manager.ConnState(conn, http.StateIdle)

	assert.Eventually(t, notified.Load, time.Second, time.Millisecond)
}

func TestConnectionPressureManager_ClosesOldestIdleConnection(t *testing.T) {
	recorder := httpserverMocks.NewServerMetricRecorder(t)
	manager := httpserver.NewConnectionPressureManagerWithInterfaces(t.Context(), clock.NewFakeClock(), recorder)
	connA := httpserverMocks.NewNetConn(t)
	connB := httpserverMocks.NewNetConn(t)
	connAClosed := atomic.Bool{}
	connA.EXPECT().Close().Run(func() {
		connAClosed.Store(true)
	}).Return(nil).Once()
	recorder.EXPECT().TrackConnectionOpened(matcher.Context).Return().Twice()
	recorder.EXPECT().TrackConnectionClosed(matcher.Context).Return().Once()

	manager.ConnState(connA, http.StateNew)
	manager.ConnState(connA, http.StateIdle)
	manager.ConnState(connB, http.StateNew)
	manager.ConnState(connB, http.StateIdle)

	closed, err := manager.CloseOldestIdleConnection()

	require.NoError(t, err)
	assert.True(t, closed)
	assert.True(t, connAClosed.Load())
}

func TestConnectionPressureManager_DoesNotCloseActiveConnection(t *testing.T) {
	recorder := httpserverMocks.NewServerMetricRecorder(t)
	manager := httpserver.NewConnectionPressureManagerWithInterfaces(t.Context(), clock.NewFakeClock(), recorder)
	conn := httpserverMocks.NewNetConn(t)
	recorder.EXPECT().TrackConnectionOpened(matcher.Context).Return().Once()

	manager.ConnState(conn, http.StateNew)

	closed, err := manager.CloseOldestIdleConnection()

	require.NoError(t, err)
	assert.False(t, closed)
}

func TestConnectionPressureManager_TracksClosedOnceWhenIdleConnectionWasAlreadyEvicted(t *testing.T) {
	recorder := httpserverMocks.NewServerMetricRecorder(t)
	manager := httpserver.NewConnectionPressureManagerWithInterfaces(t.Context(), clock.NewFakeClock(), recorder)
	conn := httpserverMocks.NewNetConn(t)
	conn.EXPECT().Close().Return(nil).Once()
	recorder.EXPECT().TrackConnectionOpened(matcher.Context).Return().Once()
	recorder.EXPECT().TrackConnectionClosed(matcher.Context).Return().Once()

	manager.ConnState(conn, http.StateNew)
	manager.ConnState(conn, http.StateIdle)

	closed, err := manager.CloseOldestIdleConnection()
	require.NoError(t, err)
	require.True(t, closed)

	manager.ConnState(conn, http.StateClosed)
}
