package email

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/emersion/go-smtp"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/uuid"
)

var (
	_ SenderWithAttachments = &smtpSender{}
	_ SmtpClient            = &smtp.Client{}
)

//go:generate go run github.com/vektra/mockery/v2 --name SmtpClient
type SmtpClient interface {
	SendMail(from string, to []string, msg io.Reader) (err error)
}

type SenderSmtpSettings struct {
	Server string `cfg:"server"`
}

type clientFactory func() (SmtpClient, error)

type smtpClientFactory func(server string) (SmtpClient, error)

type smtpSender struct {
	uuid                  uuid.Uuid
	clock                 clock.Clock
	fromAddress           string
	maxEncodedMessageSize int
	clientFactory         clientFactory
}

func NewSmtpSender(config cfg.Config, name string) (SenderWithAttachments, error) {
	return newSmtpSenderFromConfig(config, name, func(server string) (SmtpClient, error) {
		return smtp.Dial(server)
	}, uuid.New(), clock.Provider)
}

func newSmtpSenderFromConfig(config cfg.Config, name string, dial smtpClientFactory, uuid uuid.Uuid, clock clock.Clock) (SenderWithAttachments, error) {
	key := fmt.Sprintf("email.%s", name)

	smtpSettings := &SenderSmtpSettings{}
	if err := config.UnmarshalKey(key, smtpSettings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal smtp settings for key %q in NewSmtpSender: %w", key, err)
	}

	emailConfig := &emailSettings{}
	if err := config.UnmarshalKey(key, emailConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal email settings for key %q in NewSmtpSender: %w", key, err)
	}
	if err := emailConfig.validate(); err != nil {
		return nil, fmt.Errorf("invalid email settings for key %q in NewSmtpSender: %w", key, err)
	}

	clientFactory := func() (SmtpClient, error) {
		return dial(smtpSettings.Server)
	}

	// dial in the boot once to make sure the server exists and has an open port
	if _, err := clientFactory(); err != nil {
		return nil, fmt.Errorf("failed to connect to SMTP server: %v", err)
	}

	return newSmtpSender(clientFactory, uuid, clock, emailConfig.FromAddress, emailConfig.MaxEncodedMessageSize), nil
}

func NewSmtpSenderWithInterfaces(clientFactory clientFactory, uuid uuid.Uuid, clock clock.Clock, fromAddress string) SenderWithAttachments {
	return newSmtpSender(clientFactory, uuid, clock, fromAddress, defaultMaxEncodedMessageSize)
}

func newSmtpSender(clientFactory clientFactory, uuid uuid.Uuid, clock clock.Clock, fromAddress string, maxEncodedMessageSize int) SenderWithAttachments {
	return &smtpSender{
		clientFactory:         clientFactory,
		uuid:                  uuid,
		clock:                 clock,
		fromAddress:           fromAddress,
		maxEncodedMessageSize: maxEncodedMessageSize,
	}
}

func (s *smtpSender) SendEmail(ctx context.Context, email Email) error {
	if err := email.validate(); err != nil {
		return err
	}

	return s.sendEmail(ctx, EmailWithAttachments{Email: email})
}

func (s *smtpSender) SendEmailWithAttachments(ctx context.Context, email EmailWithAttachments) error {
	if err := email.validate(); err != nil {
		return err
	}

	return s.sendEmail(ctx, email)
}

func (s *smtpSender) sendEmail(_ context.Context, email EmailWithAttachments) error {
	envelope, err := parseEmailEnvelope(s.fromAddress, email.Recipients)
	if err != nil {
		return fmt.Errorf("could not parse email envelope: %w", err)
	}

	body, err := s.compileBody(email, envelope)
	if err != nil {
		return err
	}

	// We create a client every time since the connection times out after a few minutes.
	client, err := s.clientFactory()
	if err != nil {
		return fmt.Errorf("cannot dial smtp server: %w", err)
	}

	return client.SendMail(envelope.sender.Address, envelope.recipientAddresses(), body)
}

func (s *smtpSender) compileBody(email EmailWithAttachments, envelope emailEnvelope) (io.Reader, error) {
	body := &bytes.Buffer{}
	writer := &encodedMessageSizeWriter{writer: body, limit: s.maxEncodedMessageSize}
	if err := compileMIME(email, envelope, s.uuid.NewV4, s.clock.Now().UTC(), writer); err != nil {
		return nil, fmt.Errorf("could not compile email body: %w", err)
	}

	return bytes.NewReader(body.Bytes()), nil
}
