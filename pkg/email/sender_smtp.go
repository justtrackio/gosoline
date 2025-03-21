package email

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"

	"github.com/emersion/go-smtp"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

const (
	smtpLineBreak = "\r\n"
)

var (
	_ Sender     = &smtpSender{}
	_ SmtpClient = &smtp.Client{}
)

type SmtpClient interface {
	SendMail(from string, to []string, msg io.Reader) (err error)
}

type SenderSmtpSettings struct {
	Server string `cfg:"server"`
}

type smtpSender struct {
	client      SmtpClient
	fromAddress string
}

func NewSmtpSender(config cfg.Config, name string) (Sender, error) {
	key := fmt.Sprintf("email.%s", name)

	smtpSettings := &SenderSmtpSettings{}
	config.UnmarshalKey(key, smtpSettings)

	client, err := smtp.Dial(smtpSettings.Server)
	if err != nil {
		return nil, fmt.Errorf("cannot dial smtp server: %w", err)
	}

	emailSettings := &emailSettings{}
	config.UnmarshalKey(key, emailSettings)

	return NewSmtpSenderWithInterfaces(client, emailSettings.FromAddress), nil
}

func NewSmtpSenderWithInterfaces(client SmtpClient, fromAddress string) Sender {
	return &smtpSender{
		client:      client,
		fromAddress: fromAddress,
	}
}

func (s *smtpSender) SendEmail(_ context.Context, mail Mail) error {
	if mail.HtmlBody == nil && mail.TextBody == nil {
		return fmt.Errorf("email body cannot be empty")
	}

	body, err := s.compileBody(mail.Subject, mail.TextBody, mail.HtmlBody)
	if err != nil {
		return fmt.Errorf("could not compile email body: %w", err)
	}

	return s.client.SendMail(s.fromAddress, mail.Recipients, body)
}

func (s *smtpSender) compileBody(subject string, text, html *string) (io.Reader, error) {
	body := &bytes.Buffer{}

	headers := []string{
		fmt.Sprintf("Subject: %s", subject),
		"Content-Type: multipart/alternative; boundary=\"gosoMail\"",
	}

	for _, h := range headers {
		if _, err := body.WriteString(h + smtpLineBreak); err != nil {
			return nil, fmt.Errorf("could not write email header: %w", err)
		}
	}

	if _, err := body.WriteString(smtpLineBreak); err != nil {
		return nil, fmt.Errorf("could not write email header: %w", err)
	}

	writer := multipart.NewWriter(body)
	if err := writer.SetBoundary("gosoMail"); err != nil {
		return nil, fmt.Errorf("could not write email boundary: %w", err)
	}

	if html != nil {
		htmlHeader := mimeHeader("text/html")
		htmlBody, err := writer.CreatePart(htmlHeader)
		if err != nil {
			return nil, fmt.Errorf("could not create mail header part: %w", err)
		}

		htmlBytes := []byte(mdl.EmptyIfNil(html) + smtpLineBreak)
		if _, err := htmlBody.Write(htmlBytes); err != nil {
			return nil, fmt.Errorf("could not write text body: %w", err)
		}
	}

	if text != nil {
		textHeader := mimeHeader("text/plain")
		textBody, err := writer.CreatePart(textHeader)
		if err != nil {
			return nil, fmt.Errorf("could not create mail header part: %w", err)
		}

		textBytes := []byte(mdl.EmptyIfNil(text) + smtpLineBreak)
		if _, err := textBody.Write(textBytes); err != nil {
			return nil, fmt.Errorf("could not write text body: %w", err)
		}
	}

	body.WriteString(smtpLineBreak + "--gosoMail--" + smtpLineBreak)

	return body, nil
}

func mimeHeader(contentType string) textproto.MIMEHeader {
	return textproto.MIMEHeader{
		"Content-Type":              []string{contentType + "; charset=\"utf-8\""},
		"Content-Transfer-Encoding": []string{"quoted-printable"},
		"Content-Disposition":       []string{"inline"},
	}
}
