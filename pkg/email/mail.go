package email

type Mail struct {
	Recipients []string
	Subject    string
	TextBody   *string
	HtmlBody   *string
}
