package httpserver_test

import (
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/httpserver"
	httpserverMocks "github.com/justtrackio/gosoline/pkg/httpserver/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectionLimit_DisabledReturnsOriginalListener(t *testing.T) {
	listener := httpserverMocks.NewNetListener(t)
	manager := httpserverMocks.NewConnectionPressureManager(t)
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))

	limited := httpserver.NewConnectionLimitListener(
		t.Context(),
		logger,
		listener,
		httpserver.ConcurrencySettings{},
		manager,
	)

	assert.Same(t, listener, limited)
}

func TestConnectionLimit_AcceptWaitsUntilConnectionSlotIsReleased(t *testing.T) {
	listener := httpserverMocks.NewNetListener(t)
	idleAvailable := make(chan struct{})
	manager := httpserverMocks.NewConnectionPressureManager(t)
	manager.EXPECT().CloseOldestIdleConnection().Return(false, nil).Maybe()
	manager.EXPECT().IdleConnectionAvailable().Return(idleAvailable).Maybe()
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))

	limited := httpserver.NewConnectionLimitListener(
		t.Context(),
		logger,
		listener,
		httpserver.ConcurrencySettings{MaxConnections: 1},
		manager,
	)

	connA := httpserverMocks.NewNetConn(t)
	connA.EXPECT().Close().Return(nil).Once()
	listener.EXPECT().Accept().Return(connA, nil).Once()
	limitedConnA, err := limited.Accept()
	require.NoError(t, err)
	require.NotNil(t, limitedConnA)

	acceptedSecond := atomic.Bool{}
	listener.EXPECT().Accept().Return(httpserverMocks.NewNetConn(t), nil).Once()
	go func() {
		connB, acceptErr := limited.Accept()
		require.NoError(t, acceptErr)
		require.NotNil(t, connB)
		acceptedSecond.Store(true)
	}()

	assert.Never(t, acceptedSecond.Load, 20*time.Millisecond, time.Millisecond)

	require.NoError(t, limitedConnA.Close())
	assert.Eventually(t, acceptedSecond.Load, time.Second, time.Millisecond)
}

func TestConnectionLimit_ClosesIdleConnectionBeforeWaiting(t *testing.T) {
	listener := httpserverMocks.NewNetListener(t)
	idleAvailable := make(chan struct{})
	closedIdle := atomic.Bool{}
	manager := httpserverMocks.NewConnectionPressureManager(t)
	manager.EXPECT().CloseOldestIdleConnection().Run(func() {
		closedIdle.Store(true)
	}).Return(true, nil).Once()
	manager.EXPECT().CloseOldestIdleConnection().Return(false, nil).Maybe()
	manager.EXPECT().IdleConnectionAvailable().Return(idleAvailable).Maybe()
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))

	limited := httpserver.NewConnectionLimitListener(
		t.Context(),
		logger,
		listener,
		httpserver.ConcurrencySettings{MaxConnections: 1},
		manager,
	)

	connA := httpserverMocks.NewNetConn(t)
	connA.EXPECT().Close().Return(nil).Once()
	listener.EXPECT().Accept().Return(connA, nil).Once()
	limitedConnA, err := limited.Accept()
	require.NoError(t, err)
	require.NotNil(t, limitedConnA)

	listener.EXPECT().Accept().Return(httpserverMocks.NewNetConn(t), nil).Once()
	acceptedSecond := atomic.Bool{}
	go func() {
		connB, acceptErr := limited.Accept()
		require.NoError(t, acceptErr)
		require.NotNil(t, connB)
		acceptedSecond.Store(true)
	}()

	assert.Eventually(t, closedIdle.Load, time.Second, time.Millisecond)
	assert.Never(t, acceptedSecond.Load, 20*time.Millisecond, time.Millisecond)

	require.NoError(t, limitedConnA.Close())
	assert.Eventually(t, acceptedSecond.Load, time.Second, time.Millisecond)
}

func TestConnectionLimit_WakesUpWhenIdleConnectionBecomesAvailable(t *testing.T) {
	listener := httpserverMocks.NewNetListener(t)
	idleAvailable := make(chan struct{})
	manager := httpserverMocks.NewConnectionPressureManager(t)
	manager.EXPECT().CloseOldestIdleConnection().Return(false, nil).Maybe()
	manager.EXPECT().IdleConnectionAvailable().Return(idleAvailable).Maybe()
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))

	limited := httpserver.NewConnectionLimitListener(
		t.Context(),
		logger,
		listener,
		httpserver.ConcurrencySettings{MaxConnections: 1},
		manager,
	)

	connA := httpserverMocks.NewNetConn(t)
	connA.EXPECT().Close().Return(nil).Once()
	listener.EXPECT().Accept().Return(connA, nil).Once()
	limitedConnA, err := limited.Accept()
	require.NoError(t, err)
	require.NotNil(t, limitedConnA)

	listener.EXPECT().Accept().Return(httpserverMocks.NewNetConn(t), nil).Once()
	acceptedSecond := atomic.Bool{}
	go func() {
		connB, acceptErr := limited.Accept()
		require.NoError(t, acceptErr)
		require.NotNil(t, connB)
		acceptedSecond.Store(true)
	}()

	assert.Never(t, acceptedSecond.Load, 20*time.Millisecond, time.Millisecond)

	require.NoError(t, limitedConnA.Close())
	close(idleAvailable)
	assert.Eventually(t, acceptedSecond.Load, time.Second, time.Millisecond)
}

func TestConnectionLimit_CloseUnblocksWaitingAccept(t *testing.T) {
	listener := httpserverMocks.NewNetListener(t)
	idleAvailable := make(chan struct{})
	manager := httpserverMocks.NewConnectionPressureManager(t)
	manager.EXPECT().CloseOldestIdleConnection().Return(false, nil).Maybe()
	manager.EXPECT().IdleConnectionAvailable().Return(idleAvailable).Maybe()
	listener.EXPECT().Close().Return(nil).Once()
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))

	limited := httpserver.NewConnectionLimitListener(
		t.Context(),
		logger,
		listener,
		httpserver.ConcurrencySettings{MaxConnections: 1},
		manager,
	)

	listener.EXPECT().Accept().Return(httpserverMocks.NewNetConn(t), nil).Once()
	connA, err := limited.Accept()
	require.NoError(t, err)
	require.NotNil(t, connA)

	acceptedSecond := make(chan error, 1)
	acceptReturned := atomic.Bool{}
	go func() {
		_, acceptErr := limited.Accept()
		acceptReturned.Store(true)
		acceptedSecond <- acceptErr
	}()

	assert.Never(t, acceptReturned.Load, 20*time.Millisecond, time.Millisecond)

	require.NoError(t, limited.Close())
	require.Eventually(t, acceptReturned.Load, time.Second, time.Millisecond)
	assert.ErrorIs(t, <-acceptedSecond, net.ErrClosed)
}
