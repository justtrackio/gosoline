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

var (
	_ Sender = &sesSender{}
)

type SenderSesSettings struct {
	ClientName string `cfg:"client_name" default:"default"`
}

type sesSender struct {
	logger log.Logger
	client gosoSES.Client

	fromAddress string
}

func NewSesSender(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Sender, error) {
	key := fmt.Sprintf("email.%s", name)

	sesSettings := &SenderSesSettings{}
	config.UnmarshalKey(key, sesSettings)

	sesClient, err := gosoSES.ProvideClient(ctx, config, logger, sesSettings.ClientName)
	if err != nil {
		return nil, fmt.Errorf("can not create ses client with name %s: %w", sesSettings.ClientName, err)
	}

	emailSettings := &emailSettings{}
	config.UnmarshalKey(key, emailSettings)

	return NewSesSenderWithInterfaces(logger, sesClient, emailSettings.FromAddress), nil
}

func NewSesSenderWithInterfaces(logger log.Logger, client gosoSES.Client, fromAddress string) Sender {
	return &sesSender{
		logger:      logger,
		client:      client,
		fromAddress: fromAddress,
	}
}

func (s *sesSender) SendEmail(ctx context.Context, mail Mail) error {
	body := &types.Body{}

	if mail.HtmlBody != nil {
		body.Html = &types.Content{Data: mail.HtmlBody, Charset: aws.String("UTF-8")}
	}

	if mail.TextBody != nil {
		body.Text = &types.Content{Data: mail.TextBody, Charset: aws.String("UTF-8")}
	}

	if body.Html == nil && body.Text == nil {
		return fmt.Errorf("email body cannot be empty")
	}

	input := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(s.fromAddress),
		Destination: &types.Destination{
			ToAddresses: mail.Recipients,
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: aws.String(mail.Subject), Charset: aws.String("UTF-8")},
				Body:    body,
			},
		},
	}

	_, err := s.client.SendEmail(ctx, input)

	return err
}
