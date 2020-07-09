package cfg

type Logger interface {
	Infof(msg string, args ...interface{})
	Errorf(err error, msg string, args ...interface{})
	Fatalf(err error, msg string, args ...interface{})
}
