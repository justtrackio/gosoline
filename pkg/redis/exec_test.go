package redis_test

import (
	"errors"
	"fmt"
	"io"
	"net"
	"testing"

	"github.com/justtrackio/gosoline/pkg/redis"
	baseRedis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "io.EOF",
			err:      io.EOF,
			expected: true,
		},
		{
			name:     "wrapped io.EOF",
			err:      fmt.Errorf("connection failed: %w", io.EOF),
			expected: true,
		},
		{
			name:     "ErrPoolTimeout",
			err:      baseRedis.ErrPoolTimeout,
			expected: true,
		},
		{
			name:     "wrapped ErrPoolTimeout",
			err:      fmt.Errorf("redis operation failed: %w", baseRedis.ErrPoolTimeout),
			expected: true,
		},
		{
			name:     "ErrPoolExhausted",
			err:      baseRedis.ErrPoolExhausted,
			expected: true,
		},
		{
			name:     "wrapped ErrPoolExhausted",
			err:      fmt.Errorf("redis operation failed: %w", baseRedis.ErrPoolExhausted),
			expected: true,
		},
		{
			name:     "net.Error (timeout)",
			err:      &net.DNSError{IsTimeout: true},
			expected: true,
		},
		{
			name:     "ERR max number of clients reached",
			err:      errors.New("ERR max number of clients reached"),
			expected: true,
		},
		{
			name:     "LOADING message",
			err:      errors.New("LOADING Redis is loading the dataset in memory"),
			expected: true,
		},
		{
			name:     "READONLY message",
			err:      errors.New("READONLY You can't write against a read only replica"),
			expected: true,
		},
		{
			name:     "CLUSTERDOWN message",
			err:      errors.New("CLUSTERDOWN Hash slot not served"),
			expected: true,
		},
		{
			name:     "random error",
			err:      errors.New("some random error"),
			expected: false,
		},
		{
			name:     "wrapped random error",
			err:      fmt.Errorf("operation failed: %w", errors.New("random error")),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redis.IsRetryableError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
