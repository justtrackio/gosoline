package cfg

import "context"

type Logger interface {
	Info(ctx context.Context, format string, args ...any)
	Error(ctx context.Context, format string, args ...any)
}
