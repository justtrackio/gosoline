package email

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"

	"github.com/emersion/go-smtp"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/uuid"
)

const (
	smtpLineBreak = "\r\n"
)

var (
	_ Sender     = &smtpSender{}
	_ SmtpClient = &smtp.Client{}
)

//go:generate mockery --name SmtpClient
type SmtpClient interface {
	SendMail(from string, to []string, msg io.Reader) (err error)
}

type SenderSmtpSettings struct {
	Server string `cfg:"server"`
}

type smtpSender struct {
	client      SmtpClient
	uuid        uuid.Uuid
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

	uuid := uuid.New()

	return NewSmtpSenderWithInterfaces(client, uuid, emailSettings.FromAddress), nil
}

func NewSmtpSenderWithInterfaces(client SmtpClient, uuid uuid.Uuid, fromAddress string) Sender {
	return &smtpSender{
		client:      client,
		uuid:        uuid,
		fromAddress: fromAddress,
	}
}

func (s *smtpSender) SendEmail(_ context.Context, email Email) error {
	if email.HtmlBody == nil && email.TextBody == nil {
		return fmt.Errorf("email body cannot be empty")
	}

	body, err := s.compileBody(email.Subject, email.TextBody, email.HtmlBody)
	if err != nil {
		return fmt.Errorf("could not compile email body: %w", err)
	}

	return s.client.SendMail(s.fromAddress, email.Recipients, body)
}

func (s *smtpSender) compileBody(subject string, text, html *string) (io.Reader, error) {
	body := &bytes.Buffer{}

	boundary := s.uuid.NewV4()

	subjectBytes, err := encodeQuotedPrintable(subject)
	if err != nil {
		return nil, err
	}

	headers := []string{
		fmt.Sprintf("Subject: %s", subjectBytes) + smtpLineBreak,
		fmt.Sprintf("Content-Type: multipart/alternative; boundary=%q", boundary) + smtpLineBreak,
		smtpLineBreak,
	}

	for _, h := range headers {
		if _, err := body.WriteString(h); err != nil {
			return nil, fmt.Errorf("could not write email header: %w", err)
		}
	}

	writer := multipart.NewWriter(body)
	if err := writer.SetBoundary(boundary); err != nil {
		return nil, fmt.Errorf("could not write email boundary: %w", err)
	}

	if html != nil {
		htmlHeader := mimeHeader("text/html")
		htmlBody, err := writer.CreatePart(htmlHeader)
		if err != nil {
			return nil, fmt.Errorf("could not create email header part: %w", err)
		}

		htmlBytes, err := encodeQuotedPrintable(mdl.EmptyIfNil(html) + smtpLineBreak)
		if err != nil {
			return nil, fmt.Errorf("could not encode html body: %w", err)
		}

		if _, err := htmlBody.Write(htmlBytes); err != nil {
			return nil, fmt.Errorf("could not write text body: %w", err)
		}
	}

	if text != nil {
		textHeader := mimeHeader("text/plain")
		textBody, err := writer.CreatePart(textHeader)
		if err != nil {
			return nil, fmt.Errorf("could not create email header part: %w", err)
		}

		textBytes, err := encodeQuotedPrintable(mdl.EmptyIfNil(text) + smtpLineBreak)
		if err != nil {
			return nil, fmt.Errorf("could not encode text body: %w", err)
		}

		if _, err := textBody.Write(textBytes); err != nil {
			return nil, fmt.Errorf("could not write text body: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("could not close multipart writer: %w", err)
	}

	return body, nil
}

func encodeQuotedPrintable(data string) ([]byte, error) {
	quoted := &bytes.Buffer{}

	writer := quotedprintable.NewWriter(quoted)
	if _, err := writer.Write([]byte(data)); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return quoted.Bytes(), nil
}

func mimeHeader(contentType string) textproto.MIMEHeader {
	return textproto.MIMEHeader{
		"Content-Type":              []string{contentType + "; charset=\"utf-8\""},
		"Content-Transfer-Encoding": []string{"quoted-printable"},
		"Content-Disposition":       []string{"inline"},
	}
}
