package consumer

import "github.com/twmb/franz-go/pkg/kgo"

//go:generate go run github.com/vektra/mockery/v2 --name KafkaMessageHandler
type KafkaMessageHandler interface {
	Handle(messages []*kgo.Record)
	Stop()
}
