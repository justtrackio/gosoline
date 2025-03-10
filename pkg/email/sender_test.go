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

type SenderTestSuite struct {
	suite.Suite

	sender email.Sender

	logger *loggerMocks.Logger
	client *mocks.Client
	ctx    context.Context
}

func TestSenderTestSuite(t *testing.T) {
	suite.Run(t, new(SenderTestSuite))
}

func (s *SenderTestSuite) SetupTest() {
	s.logger = new(loggerMocks.Logger)
	s.client = mocks.NewClient(s.T())

	s.sender = email.NewSenderWithInterfaces(
		s.logger,
		s.client,
		"sender@example.com",
	)

	s.ctx = context.Background()
}

func (s *SenderTestSuite) TearDownTest() {
	s.client.AssertExpectations(s.T())
}

func (s *SenderTestSuite) TestSendEmail_TextEmail() {
	recipients := []string{"recipient@example.com"}
	subject := "Test Subject"
	body := "This is a plain text email."
	htmlEmptyBody := ""

	expectedEmailInput := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String("sender@example.com"),
		Destination: &types.Destination{
			ToAddresses: recipients,
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: aws.String(subject)},
				Body: &types.Body{
					Text: &types.Content{Data: aws.String(body)},
				},
			},
		},
	}

	s.client.EXPECT().SendEmail(mock.Anything, expectedEmailInput).Return(&sesv2.SendEmailOutput{}, nil)

	err := s.sender.SendEmail(s.ctx, recipients, subject, body, htmlEmptyBody)
	s.NoError(err)
}

func (s *SenderTestSuite) TestSendEmail_HtmlEmail() {
	recipients := []string{"recipient@example.com"}
	subject := "Test Subject"
	emptyBody := ""
	htmlBody := "<h1>This is an HTML email.</h1>"

	expectedEmailInput := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String("sender@example.com"),
		Destination: &types.Destination{
			ToAddresses: recipients,
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: aws.String(subject)},
				Body: &types.Body{
					Html: &types.Content{Data: aws.String(htmlBody)},
				},
			},
		},
	}

	s.client.EXPECT().SendEmail(mock.Anything, expectedEmailInput).Return(&sesv2.SendEmailOutput{}, nil)

	err := s.sender.SendEmail(s.ctx, recipients, subject, emptyBody, htmlBody)
	s.NoError(err)
}

func (s *SenderTestSuite) TestSendEmail_MultiFormatEmail() {
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
				Subject: &types.Content{Data: aws.String(subject)},
				Body: &types.Body{
					Text: &types.Content{Data: aws.String(body)},
					Html: &types.Content{Data: aws.String(htmlBody)},
				},
			},
		},
	}

	s.client.EXPECT().SendEmail(mock.Anything, expectedEmailInput).Return(&sesv2.SendEmailOutput{}, nil)

	err := s.sender.SendEmail(s.ctx, recipients, subject, body, htmlBody)
	s.NoError(err)
}

func (s *SenderTestSuite) TestSendEmail_NoBodyProvided() {
	recipients := []string{"recipient@example.com"}
	subject := "Test Subject"
	body := ""
	htmlBody := ""

	err := s.sender.SendEmail(s.ctx, recipients, subject, body, htmlBody)

	s.Error(err)
	s.EqualError(err, "email body cannot be empty")
}

func (s *SenderTestSuite) TestSendEmail_ErrorFromSES() {
	recipients := []string{"recipient@example.com"}
	subject := "Test Error Handling"
	body := "This email should trigger an error."
	htmlEmptyBody := ""

	expectedEmailInput := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String("sender@example.com"),
		Destination: &types.Destination{
			ToAddresses: recipients,
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: aws.String(subject)},
				Body: &types.Body{
					Text: &types.Content{Data: aws.String(body)},
				},
			},
		},
	}
	s.client.EXPECT().SendEmail(mock.Anything, expectedEmailInput).Return(nil, errors.New("error"))

	err := s.sender.SendEmail(s.ctx, recipients, subject, body, htmlEmptyBody)
	s.Error(err)
}
