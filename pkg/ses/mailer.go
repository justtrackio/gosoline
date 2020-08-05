package ses

import (
	"context"
	"github.com/applike/gosoline/pkg/cast"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/applike/gosoline/pkg/encoding/json"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/aws/aws-sdk-go/service/ses/sesiface"
)

//go:generate mockery -name SimpleMailer
type SimpleMailer interface {
	Send(ctx context.Context, message Message) error
}

//go:generate mockery -name TemplatedMailer
type TemplatedMailer interface {
	Send(ctx context.Context, message TemplatedMessage) error
}

type Settings struct {
	Client  cloud.ClientSettings
	Backoff cloud.BackoffSettings
}

type simpleMailer struct {
	logger   mon.Logger
	client   sesiface.SESAPI
	settings *Settings
}

type templatedMailer struct {
	logger   mon.Logger
	client   sesiface.SESAPI
	settings *Settings
}

func NewSimpleMailer(config cfg.Config, logger mon.Logger, settings *Settings) SimpleMailer {
	client := ProvideClient(config, logger, settings)

	return NewSimpleMailerWithInterfaces(logger, client, settings)
}

func NewTemplatedMailer(config cfg.Config, logger mon.Logger, settings *Settings) TemplatedMailer {
	client := ProvideClient(config, logger, settings)

	return NewTemplatedMailerWithInterfaces(logger, client, settings)
}

func NewSimpleMailerWithInterfaces(logger mon.Logger, client sesiface.SESAPI, s *Settings) SimpleMailer {
	return &simpleMailer{
		logger:   logger,
		client:   client,
		settings: s,
	}
}

func NewTemplatedMailerWithInterfaces(logger mon.Logger, client sesiface.SESAPI, s *Settings) TemplatedMailer {
	return &templatedMailer{
		logger:   logger,
		client:   client,
		settings: s,
	}
}

func (e *simpleMailer) Send(ctx context.Context, message Message) error {
	input := &ses.SendEmailInput{
		Destination: getDestination(message.Recipients),
		Message: &ses.Message{
			Body: &ses.Body{
				Html: &ses.Content{
					Data: aws.String(message.HtmlMessage),
				},
				Text: &ses.Content{
					Data: aws.String(message.TextMessage),
				},
			},
			Subject: &ses.Content{
				Data: aws.String(message.Subject),
			},
		},
		Source: aws.String(message.From),
	}

	_, err := e.client.SendEmailWithContext(ctx, input)

	if cloud.IsRequestCanceled(err) {
		e.logger.Info("request was canceled while sending email")

		return err
	}

	if err != nil {
		e.logger.Error(err, "could not send email")
	}

	return err
}

func getDestination(r Recipients) *ses.Destination {
	return &ses.Destination{
		ToAddresses:  cast.ToSlicePtrString(r.To),
		CcAddresses:  cast.ToSlicePtrString(r.Cc),
		BccAddresses: cast.ToSlicePtrString(r.Bcc),
	}
}

func (e *templatedMailer) Send(ctx context.Context, message TemplatedMessage) error {
	td, err := json.Marshal(message.TemplateData)
	if err != nil {
		return err
	}

	ctx = mon.AppendLoggerContextField(ctx, mon.Fields{
		"template_name": message.TemplateName,
	})
	logger := e.logger.WithContext(ctx)

	input := &ses.SendTemplatedEmailInput{
		Destination:  getDestination(message.Recipients),
		Source:       aws.String(message.From),
		Template:     aws.String(message.TemplateName),
		TemplateData: aws.String(string(td)),
	}

	_, err = e.client.SendTemplatedEmailWithContext(ctx, input)

	if cloud.IsRequestCanceled(err) {
		logger.Info("request was canceled while sending email")

		return err
	}

	if err != nil {
		e.logger.Error(err, "could not send email")
	}

	return err
}
