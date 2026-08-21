package email

import (
	"bytes"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	gosoSES "github.com/justtrackio/gosoline/pkg/cloud/aws/ses"
	"github.com/justtrackio/gosoline/pkg/log"
)

var _ SenderWithAttachments = &sesSender{}

type SenderSesSettings struct {
	ClientName string `cfg:"client_name" default:"default"`
}

type sesSender struct {
	logger log.Logger
	client gosoSES.Client
	clock  clock.Clock

	fromAddress           string
	maxEncodedMessageSize int
}

func NewSesSender(ctx context.Context, config cfg.Config, logger log.Logger, name string) (SenderWithAttachments, error) {
	key := fmt.Sprintf("email.%s", name)

	sesSettings := &SenderSesSettings{}
	if err := config.UnmarshalKey(key, sesSettings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ses settings for key %q in NewSesSender: %w", key, err)
	}

	emailConfig := &emailSettings{}
	if err := config.UnmarshalKey(key, emailConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal email settings for key %q in NewSesSender: %w", key, err)
	}
	if err := emailConfig.validate(); err != nil {
		return nil, fmt.Errorf("invalid email settings for key %q in NewSesSender: %w", key, err)
	}

	sesClient, err := gosoSES.ProvideClient(ctx, config, logger, sesSettings.ClientName)
	if err != nil {
		return nil, fmt.Errorf("can not create ses client with name %s: %w", sesSettings.ClientName, err)
	}

	return newSesSender(logger, sesClient, clock.Provider, emailConfig.FromAddress, emailConfig.MaxEncodedMessageSize), nil
}

func NewSesSenderWithInterfaces(logger log.Logger, client gosoSES.Client, clock clock.Clock, fromAddress string) SenderWithAttachments {
	return newSesSender(logger, client, clock, fromAddress, defaultMaxEncodedMessageSize)
}

func newSesSender(logger log.Logger, client gosoSES.Client, clock clock.Clock, fromAddress string, maxEncodedMessageSize int) SenderWithAttachments {
	return &sesSender{
		logger:                logger,
		client:                client,
		clock:                 clock,
		fromAddress:           fromAddress,
		maxEncodedMessageSize: maxEncodedMessageSize,
	}
}

func (s *sesSender) SendEmail(ctx context.Context, email Email) error {
	if err := email.validate(); err != nil {
		return err
	}

	return s.sendEmail(ctx, EmailWithAttachments{Email: email})
}

func (s *sesSender) SendEmailWithAttachments(ctx context.Context, email EmailWithAttachments) error {
	if err := email.validate(); err != nil {
		return err
	}

	return s.sendEmail(ctx, email)
}

func (s *sesSender) sendEmail(ctx context.Context, email EmailWithAttachments) error {
	envelope, err := parseEmailEnvelope(s.fromAddress, email.Recipients)
	if err != nil {
		return fmt.Errorf("could not parse email envelope: %w", err)
	}

	input, err := s.emailInput(email, envelope)
	if err != nil {
		return err
	}

	_, err = s.client.SendEmail(ctx, input)

	return err
}

func (s *sesSender) emailInput(email EmailWithAttachments, envelope emailEnvelope) (*sesv2.SendEmailInput, error) {
	if len(email.Attachments) == 0 {
		return &sesv2.SendEmailInput{
			FromEmailAddress: aws.String(envelope.senderMailbox()),
			Destination:      &types.Destination{ToAddresses: envelope.recipientMailboxes()},
			Content:          simpleEmailContent(email.Email),
		}, nil
	}

	body := &bytes.Buffer{}
	writer := &encodedMessageSizeWriter{writer: body, limit: s.maxEncodedMessageSize}
	if err := compileMIME(email, envelope, generatedMIMEBoundary, s.clock.Now().UTC(), writer); err != nil {
		return nil, fmt.Errorf("could not compile email body: %w", err)
	}

	return &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(envelope.sender.Address),
		Destination:      &types.Destination{ToAddresses: envelope.recipientAddresses()},
		Content:          &types.EmailContent{Raw: &types.RawMessage{Data: body.Bytes()}},
	}, nil
}

func simpleEmailContent(email Email) *types.EmailContent {
	body := &types.Body{}
	if email.HtmlBody != nil {
		body.Html = &types.Content{Data: email.HtmlBody, Charset: aws.String("UTF-8")}
	}
	if email.TextBody != nil {
		body.Text = &types.Content{Data: email.TextBody, Charset: aws.String("UTF-8")}
	}

	return &types.EmailContent{
		Simple: &types.Message{
			Subject: &types.Content{Data: aws.String(email.Subject), Charset: aws.String("UTF-8")},
			Body:    body,
		},
	}
}
