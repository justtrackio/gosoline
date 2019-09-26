package mon

//go:generate mockery -name LoggerHook
type LoggerHook interface {
	Fire(level string, msg string, err error, data *Metadata) error
}
