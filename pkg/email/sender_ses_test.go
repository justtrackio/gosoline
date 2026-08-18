package email_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/ses/mocks"
	"github.com/justtrackio/gosoline/pkg/email"
	loggerMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
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

	s.ctx = s.T().Context()
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

	s.client.EXPECT().SendEmail(matcher.Context, expectedEmailInput).Return(&sesv2.SendEmailOutput{}, nil)

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

	s.client.EXPECT().SendEmail(matcher.Context, expectedEmailInput).Return(&sesv2.SendEmailOutput{}, nil)

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

	s.client.EXPECT().SendEmail(matcher.Context, expectedEmailInput).Return(&sesv2.SendEmailOutput{}, nil)

	email := email.Email{
		Recipients: recipients,
		Subject:    subject,
		TextBody:   &body,
		HtmlBody:   &htmlBody,
	}

	err := s.sender.SendEmail(s.ctx, email)
	s.NoError(err)
}

func (s *sesSenderTestSuite) TestSendEmail_WithAttachment() {
	recipients := []string{"recipient@example.com"}
	subject := "Your résumé"
	textBody := "Your invoice is attached."
	htmlBody := "<p>Your invoice is attached.</p>"
	attachment := &email.Attachment{
		Filename:    "invoice.pdf",
		ContentType: "application/pdf",
		Content:     []byte("%PDF-1.7\ninvoice"),
	}

	s.client.EXPECT().SendEmail(matcher.Context, mock.MatchedBy(func(input *sesv2.SendEmailInput) bool {
		s.Equal("sender@example.com", aws.ToString(input.FromEmailAddress))
		s.Equal(recipients, input.Destination.ToAddresses)
		s.Nil(input.Content.Simple)
		s.NotNil(input.Content.Raw)
		assertAttachmentMessage(s.T(), bytes.NewReader(input.Content.Raw.Data), "sender@example.com", recipients, subject, textBody, htmlBody, attachment)

		return true
	})).Return(&sesv2.SendEmailOutput{}, nil)

	err := s.sender.SendEmail(s.ctx, email.Email{
		Recipients: recipients,
		Subject:    subject,
		TextBody:   &textBody,
		HtmlBody:   &htmlBody,
		Attachment: attachment,
	})
	s.NoError(err)
}

func (s *sesSenderTestSuite) TestSendEmail_NoBodyProvided() {
	err := s.sender.SendEmail(s.ctx, email.Email{})

	s.Error(err)
	s.EqualError(err, "email body cannot be empty")
}

func (s *sesSenderTestSuite) TestSendEmail_InvalidAttachment() {
	body := "This is a plain text email."

	testCases := []struct {
		name       string
		attachment *email.Attachment
		expected   string
	}{
		{
			name:       "missing filename",
			attachment: &email.Attachment{ContentType: "application/pdf", Content: []byte("content")},
			expected:   "email attachment filename cannot be empty",
		},
		{
			name:       "missing content type",
			attachment: &email.Attachment{Filename: "invoice.pdf", Content: []byte("content")},
			expected:   "email attachment content type cannot be empty",
		},
		{
			name:       "invalid content type",
			attachment: &email.Attachment{Filename: "invoice.pdf", ContentType: "application", Content: []byte("content")},
			expected:   "email attachment content type is invalid",
		},
		{
			name:       "missing content",
			attachment: &email.Attachment{Filename: "invoice.pdf", ContentType: "application/pdf"},
			expected:   "email attachment content cannot be empty",
		},
	}

	for _, testCase := range testCases {
		s.Run(testCase.name, func() {
			err := s.sender.SendEmail(s.ctx, email.Email{TextBody: &body, Attachment: testCase.attachment})

			s.EqualError(err, testCase.expected)
		})
	}
}

func (s *sesSenderTestSuite) TestSendEmail_InvalidSubject() {
	body := "This is a plain text email."

	err := s.sender.SendEmail(s.ctx, email.Email{
		Subject:  "Invoice\r\nBcc: attacker@example.com",
		TextBody: &body,
	})

	s.EqualError(err, "email subject cannot contain line breaks")
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
	s.client.EXPECT().SendEmail(matcher.Context, expectedEmailInput).Return(nil, errors.New("error"))

	email := email.Email{
		Recipients: recipients,
		Subject:    subject,
		TextBody:   &body,
	}

	err := s.sender.SendEmail(s.ctx, email)
	s.Error(err)
}
