package stream

//go:generate mockery -name Input
type Input interface {
	Run() error
	Stop()
	Data() chan *Message
}

//go:generate mockery -name AcknowledgeableInput
type AcknowledgeableInput interface {
	Ack(msg *Message) error
	AckBatch(msgs []*Message) error
}
