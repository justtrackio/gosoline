package email

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoSES "github.com/justtrackio/gosoline/pkg/cloud/aws/ses"
	"github.com/justtrackio/gosoline/pkg/log"
)

type Settings struct {
	cfg.AppId
	ClientName  string `cfg:"client_name" default:"default"`
	FromAddress string `cfg:"from_address"`
}

//go:generate mockery --name Sender
type Sender interface {
	SendEmail(ctx context.Context, recipients []string, subject string, plaintextBody string, htmlBody string) error
}

type sesSender struct {
	logger log.Logger
	client gosoSES.Client

	fromAddress string
}

func NewSender(ctx context.Context, config cfg.Config, logger log.Logger, fromAddress, name string) (Sender, error) {
	settings := getSenderSettings(config, name)

	sesClient, err := gosoSES.ProvideClient(ctx, config, logger, settings.ClientName)
	if err != nil {
		return nil, fmt.Errorf("can not create ses client with name %s: %w", settings.ClientName, err)
	}

	return NewSenderWithInterfaces(logger, sesClient, fromAddress), nil
}

func NewSenderWithInterfaces(logger log.Logger, client gosoSES.Client, fromAddress string) Sender {
	return &sesSender{
		logger:      logger,
		client:      client,
		fromAddress: fromAddress,
	}
}

func (s *sesSender) SendEmail(ctx context.Context, recipients []string, subject string, plaintextBody string, htmlBody string) error {
	body := &types.Body{}

	if htmlBody != "" {
		body.Html = &types.Content{Data: aws.String(htmlBody)}
	}
	if plaintextBody != "" {
		body.Text = &types.Content{Data: aws.String(plaintextBody)}
	}

	if body.Html == nil && body.Text == nil {
		return fmt.Errorf("email body cannot be empty")
	}

	input := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(s.fromAddress),
		Destination: &types.Destination{
			ToAddresses: recipients,
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: aws.String(subject)},
				Body:    body,
			},
		},
	}

	_, err := s.client.SendEmail(ctx, input)

	return err
}

func getSenderSettings(config cfg.Config, name string) *Settings {
	settings := &Settings{}
	key := fmt.Sprintf("email.%s", name)
	config.UnmarshalKey(key, settings)
	settings.AppId.PadFromConfig(config)

	return settings
}
