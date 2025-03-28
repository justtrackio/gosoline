package email_test

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/ses/mocks"
	"github.com/justtrackio/gosoline/pkg/email"
	loggerMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type sesSenderTestSuite struct {
	suite.Suite

	sender email.Sender

	logger *loggerMocks.Logger
	client *mocks.Client
	ctx    context.Context
}

func TestRunSesSenderTestSuite(t *testing.T) {
	suite.Run(t, new(sesSenderTestSuite))
}

func (s *sesSenderTestSuite) SetupTest() {
	s.logger = new(loggerMocks.Logger)
	s.client = mocks.NewClient(s.T())

	s.sender = email.NewSesSenderWithInterfaces(
		s.logger,
		s.client,
		"sender@example.com",
	)

	s.ctx = context.Background()
}

func (s *sesSenderTestSuite) TestSendEmail_TextEmail() {
	recipients := []string{"recipient@example.com"}
	subject := "Test Subject"
	body := "This is a plain text email."

	expectedEmailInput := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String("sender@example.com"),
		Destination: &types.Destination{
			ToAddresses: recipients,
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: aws.String(subject), Charset: aws.String("UTF-8")},
				Body: &types.Body{
					Text: &types.Content{Data: aws.String(body), Charset: aws.String("UTF-8")},
				},
			},
		},
	}

	s.client.EXPECT().SendEmail(mock.Anything, expectedEmailInput).Return(&sesv2.SendEmailOutput{}, nil)

	email := email.Email{
		Recipients: recipients,
		Subject:    subject,
		TextBody:   &body,
	}

	err := s.sender.SendEmail(s.ctx, email)
	s.NoError(err)
}

func (s *sesSenderTestSuite) TestSendEmail_HtmlEmail() {
	recipients := []string{"recipient@example.com"}
	subject := "Test Subject"
	htmlBody := "<h1>This is an HTML email.</h1>"

	expectedEmailInput := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String("sender@example.com"),
		Destination: &types.Destination{
			ToAddresses: recipients,
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: aws.String(subject), Charset: aws.String("UTF-8")},
				Body: &types.Body{
					Html: &types.Content{Data: aws.String(htmlBody), Charset: aws.String("UTF-8")},
				},
			},
		},
	}

	s.client.EXPECT().SendEmail(mock.Anything, expectedEmailInput).Return(&sesv2.SendEmailOutput{}, nil)

	email := email.Email{
		Recipients: recipients,
		Subject:    subject,
		HtmlBody:   &htmlBody,
	}

	err := s.sender.SendEmail(s.ctx, email)
	s.NoError(err)
}

func (s *sesSenderTestSuite) TestSendEmail_MultiFormatEmail() {
	recipients := []string{"recipient@example.com"}
	subject := "Test Subject"
	body := "This is a plain text email."
	htmlBody := "<h1>This is an HTML email.</h1>"

	expectedEmailInput := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String("sender@example.com"),
		Destination: &types.Destination{
			ToAddresses: recipients,
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: aws.String(subject), Charset: aws.String("UTF-8")},
				Body: &types.Body{
					Text: &types.Content{Data: aws.String(body), Charset: aws.String("UTF-8")},
					Html: &types.Content{Data: aws.String(htmlBody), Charset: aws.String("UTF-8")},
				},
			},
		},
	}

	s.client.EXPECT().SendEmail(mock.Anything, expectedEmailInput).Return(&sesv2.SendEmailOutput{}, nil)

	email := email.Email{
		Recipients: recipients,
		Subject:    subject,
		TextBody:   &body,
		HtmlBody:   &htmlBody,
	}

	err := s.sender.SendEmail(s.ctx, email)
	s.NoError(err)
}

func (s *sesSenderTestSuite) TestSendEmail_NoBodyProvided() {
	err := s.sender.SendEmail(s.ctx, email.Email{})

	s.Error(err)
	s.EqualError(err, "email body cannot be empty")
}

func (s *sesSenderTestSuite) TestSendEmail_ErrorFromSES() {
	recipients := []string{"recipient@example.com"}
	subject := "Test Error Handling"
	body := "This email should trigger an error."

	expectedEmailInput := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String("sender@example.com"),
		Destination: &types.Destination{
			ToAddresses: recipients,
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: aws.String(subject), Charset: aws.String("UTF-8")},
				Body: &types.Body{
					Text: &types.Content{Data: aws.String(body), Charset: aws.String("UTF-8")},
				},
			},
		},
	}
	s.client.EXPECT().SendEmail(mock.Anything, expectedEmailInput).Return(nil, errors.New("error"))

	email := email.Email{
		Recipients: recipients,
		Subject:    subject,
		TextBody:   &body,
	}

	err := s.sender.SendEmail(s.ctx, email)
	s.Error(err)
}
