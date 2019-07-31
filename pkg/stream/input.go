package stream

//go:generate mockery -name Input
type Input interface {
	Run() error
	Stop()
	Data() chan *Message
}

type AcknowledgeableInput interface {
	Ack(msg *Message) error
}
