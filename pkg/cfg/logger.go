package cfg

type Logger interface {
	Info(format string, args ...any)
	Error(format string, args ...any)
}
