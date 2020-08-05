package ses

type Recipients struct {
	To  []string
	Cc  []string
	Bcc []string
}

type Mail struct {
	From       string
	Recipients Recipients
}

type Message struct {
	Mail
	Subject     string
	TextMessage string
	HtmlMessage string
}

type TemplatedMessage struct {
	Mail
	TemplateName string
	TemplateData map[string]string
}
