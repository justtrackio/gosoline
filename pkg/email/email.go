package email

type Email struct {
	Recipients []string
	Subject    string
	TextBody   *string
	HtmlBody   *string
}
