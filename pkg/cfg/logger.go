package cfg

type Logger interface {
	Info(format string, args ...interface{})
	Error(err error, format string, args ...interface{})
}
