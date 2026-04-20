package httpserver_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfilingServer_BindsToLoopbackAddress(t *testing.T) {
	// Grab a free port on loopback then release it for the server to bind.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	require.NoError(t, l.Close())

	gin.SetMode(gin.TestMode)
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	router := gin.New()

	settings := &httpserver.ProfilingSettings{
		Api: httpserver.ProfilingApiSettings{Port: port},
	}

	profiling := httpserver.NewProfilingWithInterfaces(logger, router, settings)

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- profiling.Run(ctx)
	}()

	// Wait briefly for the server to start accepting connections.
	time.Sleep(50 * time.Millisecond)

	// Must be reachable on loopback; the server binds to 127.0.0.1.
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), time.Second)
	assert.NoError(t, err, "profiling server must be reachable on 127.0.0.1")
	if conn != nil {
		require.NoError(t, conn.Close())
	}

	// Cancel the context and verify clean shutdown.
	cancel()
	select {
	case runErr := <-errCh:
		assert.NoError(t, runErr, "profiling server Run must return nil on context cancel")
	case <-time.After(2 * time.Second):
		t.Fatal("profiling server did not shut down in time")
	}
}
