package sns

type Message struct {
	Type      string
	TopicArn  string
	MessageId string
	Message   string
}
