package kinesis

//go:generate mockery -name MessageHandler
type MessageHandler interface {
	Handle(rawMessage []byte) error
	Done()
}

type channelHandler struct {
	records chan []byte
}

func NewChannelHandler(records chan []byte) MessageHandler {
	return channelHandler{
		records: records,
	}
}

func (p channelHandler) Handle(rawMessage []byte) error {
	p.records <- rawMessage

	return nil
}

func (p channelHandler) Done() {
	close(p.records)
}
