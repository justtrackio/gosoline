package email

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/emersion/go-smtp"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/uuid"
)

var (
	_ Sender     = &smtpSender{}
	_ SmtpClient = &smtp.Client{}
)

//go:generate go run github.com/vektra/mockery/v2 --name SmtpClient
type SmtpClient interface {
	SendMail(from string, to []string, msg io.Reader) (err error)
}

type SenderSmtpSettings struct {
	Server string `cfg:"server"`
}

type clientFactory func() (SmtpClient, error)

type smtpSender struct {
	uuid          uuid.Uuid
	fromAddress   string
	clientFactory clientFactory
}

func NewSmtpSender(config cfg.Config, name string) (Sender, error) {
	key := fmt.Sprintf("email.%s", name)

	smtpSettings := &SenderSmtpSettings{}
	if err := config.UnmarshalKey(key, smtpSettings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal smtp settings for key %q in NewSmtpSender: %w", key, err)
	}

	emailSettings := &emailSettings{}
	if err := config.UnmarshalKey(key, emailSettings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal email settings for key %q in NewSmtpSender: %w", key, err)
	}

	clientFactory := func() (SmtpClient, error) {
		return smtp.Dial(smtpSettings.Server)
	}

	// dial in the boot once to make sure the server exists and has an open port
	if _, err := clientFactory(); err != nil {
		return nil, fmt.Errorf("failed to connect to SMTP server: %v", err)
	}

	return NewSmtpSenderWithInterfaces(clientFactory, uuid.New(), emailSettings.FromAddress), nil
}

func NewSmtpSenderWithInterfaces(clientFactory clientFactory, uuid uuid.Uuid, fromAddress string) Sender {
	return &smtpSender{
		clientFactory: clientFactory,
		uuid:          uuid,
		fromAddress:   fromAddress,
	}
}

func (s *smtpSender) SendEmail(_ context.Context, email Email) error {
	if err := email.validate(); err != nil {
		return err
	}

	var body io.Reader
	if email.Attachment != nil {
		compiled, err := compileAttachmentMIME(email, s.fromAddress, s.uuid.NewV4)
		if err != nil {
			return fmt.Errorf("could not compile email body: %w", err)
		}

		body = bytes.NewReader(compiled)
	} else {
		compiled, err := s.compileBody(email.Subject, email.TextBody, email.HtmlBody)
		if err != nil {
			return fmt.Errorf("could not compile email body: %w", err)
		}

		body = compiled
	}

	// We create a client every time since the connection times out after a few minutes.
	client, err := s.clientFactory()
	if err != nil {
		return fmt.Errorf("cannot dial smtp server: %w", err)
	}

	return client.SendMail(s.fromAddress, email.Recipients, body)
}

func (s *smtpSender) compileBody(subject string, text *string, html *string) (io.Reader, error) {
	body := &bytes.Buffer{}
	boundary := s.uuid.NewV4()

	if err := writeMessageHeaders(body, encodeHeader(subject), "multipart/alternative", boundary); err != nil {
		return nil, err
	}

	writer := multipart.NewWriter(body)
	if err := writer.SetBoundary(boundary); err != nil {
		return nil, fmt.Errorf("could not write email boundary: %w", err)
	}
	if err := writeBodyParts(writer, text, html); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("could not close multipart writer: %w", err)
	}

	return body, nil
}
