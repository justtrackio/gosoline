package stream

//go:generate mockery -name Input
type Input interface {
	Run() error
	Stop()
	Data() chan *Message
}
