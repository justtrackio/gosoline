package log

import (
	"context"
	"sync"
	"time"
)

type (
	fingersCrossedCtxKey struct{}

	fingersCrossedEntry struct {
		ctx       context.Context
		timestamp time.Time
		level     int
		msg       string
		args      []any
		err       error
		data      *Data
	}
)

type fingersCrossedScope struct {
	lck     sync.Mutex
	logger  *gosoLogger
	buffer  []fingersCrossedEntry
	flushed bool
}

func (s *fingersCrossedScope) flush() {
	s.lck.Lock()
	defer s.lck.Unlock()

	s.flushed = true

	if len(s.buffer) == 0 {
		return
	}

	if s.logger == nil {
		panic("cannot flush fingers-crossed scope without a logger")
	}

	for _, entry := range s.buffer {
		entry.data.ContextFields["fingers_crossed_flushed"] = true
		s.logger.executeHandlers(entry.ctx, entry.timestamp, entry.level, entry.msg, entry.args, entry.err, entry.data)
	}

	s.buffer = nil
}

func (s *fingersCrossedScope) shouldFlush(level int) bool {
	s.lck.Lock()
	defer s.lck.Unlock()

	return s.flushed || level >= PriorityError
}

// WithFingersCrossedScope creates a new context with a "fingers-crossed" logging scope.
// In this scope, logs are buffered and not immediately written. They are only flushed to the configured handlers
// if an error level log occurs within the scope (see Logger.log implementation) or if FlushFingersCrossedScope is called manually.
// This is useful for high-volume scenarios where detailed logs are only needed for debugging failures.
func WithFingersCrossedScope(ctx context.Context) context.Context {
	if _, ok := ctx.Value(fingersCrossedCtxKey{}).(*fingersCrossedScope); ok {
		return ctx
	}

	return context.WithValue(ctx, fingersCrossedCtxKey{}, &fingersCrossedScope{})
}

// FlushFingersCrossedScope forces the flushing of all buffered logs in the current fingers-crossed scope.
// It is a no-op if the context does not contain a fingers-crossed scope.
func FlushFingersCrossedScope(ctx context.Context) {
	scope := getFingersCrossedScope(ctx)

	if scope == nil {
		return
	}

	scope.flush()
}

func appendToFingersCrossedScope(ctx context.Context, logger *gosoLogger, timestamp time.Time, level int, msg string, args []any, loggedErr error, data *Data) bool {
	scope := getFingersCrossedScope(ctx)

	if scope == nil {
		return false
	}

	scope.lck.Lock()
	defer scope.lck.Unlock()

	scope.logger = logger
	scope.buffer = append(scope.buffer, fingersCrossedEntry{
		ctx:       ctx,
		timestamp: timestamp,
		level:     level,
		msg:       msg,
		args:      args,
		err:       loggedErr,
		data:      data,
	})

	return true
}

func getFingersCrossedScope(ctx context.Context) *fingersCrossedScope {
	scope, _ := ctx.Value(fingersCrossedCtxKey{}).(*fingersCrossedScope)

	return scope
}
