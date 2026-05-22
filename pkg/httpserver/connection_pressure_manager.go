package httpserver

import (
	"container/list"
	"context"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
)

//go:generate go run github.com/vektra/mockery/v2 --name ConnectionPressureManager
type ConnectionPressureManager interface {
	CloseOldestIdleConnection() (bool, error)
	IdleConnectionAvailable() <-chan struct{}
	ConnState(net.Conn, http.ConnState)
}

type connectionPressureManager struct {
	ctx             context.Context
	clock           clock.Clock
	metricsRecorder ServerMetricRecorder

	mu          sync.Mutex
	connections map[net.Conn]connectionState
	idle        *list.List
	idleNotify  chan struct{}
}

type connectionState struct {
	idleSince time.Time
	idleEntry *list.Element
}

func NewConnectionPressureManager(ctx context.Context, metrics ServerMetricRecorder) ConnectionPressureManager {
	return NewConnectionPressureManagerWithInterfaces(ctx, clock.Provider, metrics)
}

func NewConnectionPressureManagerWithInterfaces(
	ctx context.Context,
	clock clock.Clock,
	metricsRecorder ServerMetricRecorder,
) ConnectionPressureManager {
	return &connectionPressureManager{
		ctx:             ctx,
		clock:           clock,
		metricsRecorder: metricsRecorder,
		connections:     make(map[net.Conn]connectionState),
		idle:            list.New(),
		idleNotify:      make(chan struct{}),
	}
}

func (m *connectionPressureManager) ConnState(conn net.Conn, state http.ConnState) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch state {
	case http.StateNew:
		m.markActive(conn)
	case http.StateActive:
		m.markActive(conn)
	case http.StateIdle:
		m.markIdle(conn)
	case http.StateHijacked, http.StateClosed:
		m.remove(conn)
	}
}

func (m *connectionPressureManager) CloseOldestIdleConnection() (bool, error) {
	m.mu.Lock()
	oldestIdleConn := m.popOldestIdleConnection()
	m.mu.Unlock()

	if oldestIdleConn == nil {
		return false, nil
	}

	if err := oldestIdleConn.Close(); err != nil {
		return true, err
	}

	return true, nil
}

func (m *connectionPressureManager) IdleConnectionAvailable() <-chan struct{} {
	return m.idleNotify
}

func (m *connectionPressureManager) markActive(conn net.Conn) {
	m.removeIdle(conn)
	if _, ok := m.connections[conn]; !ok {
		m.metricsRecorder.TrackConnectionOpened(m.ctx)
	}

	m.connections[conn] = connectionState{}
}

func (m *connectionPressureManager) markIdle(conn net.Conn) {
	m.removeIdle(conn)
	m.connections[conn] = connectionState{
		idleSince: m.clock.Now(),
		idleEntry: m.idle.PushBack(conn),
	}
	m.notifyIdleConnectionAvailable()
}

func (m *connectionPressureManager) notifyIdleConnectionAvailable() {
	select {
	case m.idleNotify <- struct{}{}:
	default:
	}
}

func (m *connectionPressureManager) remove(conn net.Conn) {
	m.removeIdle(conn)
	if _, ok := m.connections[conn]; ok {
		delete(m.connections, conn)
		m.metricsRecorder.TrackConnectionClosed(m.ctx)
	}
}

func (m *connectionPressureManager) removeIdle(conn net.Conn) {
	state, ok := m.connections[conn]
	if !ok || state.idleEntry == nil {
		return
	}

	m.idle.Remove(state.idleEntry)
	state.idleEntry = nil
	m.connections[conn] = state
}

func (m *connectionPressureManager) popOldestIdleConnection() net.Conn {
	element := m.idle.Front()
	if element == nil {
		return nil
	}

	m.idle.Remove(element)
	conn := element.Value.(net.Conn)
	delete(m.connections, conn)
	m.metricsRecorder.TrackConnectionClosed(m.ctx)

	return conn
}
