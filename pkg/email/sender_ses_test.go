package email_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/ses/mocks"
	"github.com/justtrackio/gosoline/pkg/email"
	loggerMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type sesSenderTestSuite struct {
	suite.Suite

	sender email.SenderWithAttachments
	logger *loggerMocks.Logger
	client *mocks.Client
	clock  clock.Clock
	ctx    context.Context
}

func TestRunSesSenderTestSuite(t *testing.T) {
	suite.Run(t, new(sesSenderTestSuite))
}

func (s *sesSenderTestSuite) SetupTest() {
	s.logger = new(loggerMocks.Logger)
	s.client = mocks.NewClient(s.T())
	s.clock = clock.NewFakeClockAt(time.Date(2026, time.August, 21, 10, 30, 0, 0, time.UTC))
	s.sender = email.NewSesSenderWithInterfaces(s.logger, s.client, s.clock, "sender@example.com")
	s.ctx = s.T().Context()
}

func (s *sesSenderTestSuite) TestSendEmailTextEmail() {
	recipients := []string{"Recipient <recipient@example.com>"}
	subject := "Test Subject"
	body := "This is a plain text email."
	expectedInput := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String("sender@example.com"),
		Destination:      &types.Destination{ToAddresses: []string{`"Recipient" <recipient@example.com>`}},
		Content: &types.EmailContent{Simple: &types.Message{
			Subject: &types.Content{Data: aws.String(subject), Charset: aws.String("UTF-8")},
			Body:    &types.Body{Text: &types.Content{Data: aws.String(body), Charset: aws.String("UTF-8")}},
		}},
	}
	s.client.EXPECT().SendEmail(matcher.Context, expectedInput).Return(&sesv2.SendEmailOutput{}, nil)

	s.NoError(s.sender.SendEmail(s.ctx, email.Email{Recipients: recipients, Subject: subject, TextBody: &body}))
}

func (s *sesSenderTestSuite) TestSendEmailKeepsSenderDisplayName() {
	subject := "Test Subject"
	body := "This is a plain text email."
	sender := email.NewSesSenderWithInterfaces(s.logger, s.client, s.clock, "justtrack <system@justtrack.io>")
	expectedInput := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(`"justtrack" <system@justtrack.io>`),
		Destination:      &types.Destination{ToAddresses: []string{"recipient@example.com"}},
		Content: &types.EmailContent{Simple: &types.Message{
			Subject: &types.Content{Data: aws.String(subject), Charset: aws.String("UTF-8")},
			Body:    &types.Body{Text: &types.Content{Data: aws.String(body), Charset: aws.String("UTF-8")}},
		}},
	}
	s.client.EXPECT().SendEmail(matcher.Context, expectedInput).Return(&sesv2.SendEmailOutput{}, nil)

	s.NoError(sender.SendEmail(s.ctx, email.Email{Recipients: []string{"recipient@example.com"}, Subject: subject, TextBody: &body}))
}

func (s *sesSenderTestSuite) TestSendEmailHtmlEmail() {
	recipients := []string{"recipient@example.com"}
	subject := "Test Subject"
	htmlBody := "<h1>This is an HTML email.</h1>"
	expectedInput := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String("sender@example.com"),
		Destination:      &types.Destination{ToAddresses: recipients},
		Content: &types.EmailContent{Simple: &types.Message{
			Subject: &types.Content{Data: aws.String(subject), Charset: aws.String("UTF-8")},
			Body:    &types.Body{Html: &types.Content{Data: aws.String(htmlBody), Charset: aws.String("UTF-8")}},
		}},
	}
	s.client.EXPECT().SendEmail(matcher.Context, expectedInput).Return(&sesv2.SendEmailOutput{}, nil)

	s.NoError(s.sender.SendEmail(s.ctx, email.Email{Recipients: recipients, Subject: subject, HtmlBody: &htmlBody}))
}

func (s *sesSenderTestSuite) TestSendEmailWithAttachments() {
	recipients := []string{"Recipient <recipient@example.com>"}
	subject := "Your résumé"
	textBody := "Your invoice is attached."
	htmlBody := "<p>Your invoice is attached.</p>"
	attachments := []email.Attachment{
		{Filename: "invoice.pdf", ContentType: "application/pdf", Content: []byte("%PDF-1.7\ninvoice")},
		{Filename: "details.txt", ContentType: "text/plain", Content: []byte("details")},
	}
	from := "Sender <sender@example.com>"
	sender := email.NewSesSenderWithInterfaces(s.logger, s.client, s.clock, from)

	s.client.EXPECT().SendEmail(matcher.Context, mock.MatchedBy(func(input *sesv2.SendEmailInput) bool {
		s.Equal("sender@example.com", aws.ToString(input.FromEmailAddress))
		s.Equal([]string{"recipient@example.com"}, input.Destination.ToAddresses)
		s.Nil(input.Content.Simple)
		s.NotNil(input.Content.Raw)
		assertAttachmentsMessage(s.T(), bytes.NewReader(input.Content.Raw.Data), from, recipients, subject, textBody, htmlBody, attachments)

		return true
	})).Return(&sesv2.SendEmailOutput{}, nil)

	s.NoError(sender.SendEmailWithAttachments(s.ctx, email.EmailWithAttachments{
		Email:       email.Email{Recipients: recipients, Subject: subject, TextBody: &textBody, HtmlBody: &htmlBody},
		Attachments: attachments,
	}))
}

func (s *sesSenderTestSuite) TestSendEmailWithAttachmentsEmptyDoesNotCallSES() {
	body := "body"
	err := s.sender.SendEmailWithAttachments(s.ctx, email.EmailWithAttachments{Email: email.Email{TextBody: &body}})

	s.EqualError(err, "email attachments cannot be empty")
}

func (s *sesSenderTestSuite) TestSendEmailWithAttachmentsEncodedMessageExceedsDefaultLimitDoesNotCallSES() {
	body := "The attachments exceed the encoded message size limit together."
	err := s.sender.SendEmailWithAttachments(s.ctx, email.EmailWithAttachments{
		Email: email.Email{Recipients: []string{"recipient@example.com"}, TextBody: &body},
		Attachments: []email.Attachment{
			{Filename: "large-one.bin", ContentType: "application/octet-stream", Content: bytes.Repeat([]byte("a"), 4*1024*1024)},
			{Filename: "large-two.bin", ContentType: "application/octet-stream", Content: bytes.Repeat([]byte("b"), 4*1024*1024)},
		},
	})

	s.Regexp(`encoded email message size exceeds configured limit of 10485760 bytes`, err.Error())
}

func (s *sesSenderTestSuite) TestSendEmailWithAttachmentsInvalidAttachment() {
	body := "This is a plain text email."
	testCases := []struct {
		name        string
		attachments []email.Attachment
		expected    string
	}{
		{name: "empty attachments", expected: "email attachments cannot be empty"},
		{name: "missing filename", attachments: []email.Attachment{{ContentType: "application/pdf", Content: []byte("content")}}, expected: "email attachment filename cannot be empty"},
		{name: "missing content type", attachments: []email.Attachment{{Filename: "invoice.pdf", Content: []byte("content")}}, expected: "email attachment content type cannot be empty"},
		{name: "invalid content type", attachments: []email.Attachment{{Filename: "invoice.pdf", ContentType: "application", Content: []byte("content")}}, expected: "email attachment content type is invalid"},
		{name: "filename line break in later attachment", attachments: []email.Attachment{{Filename: "valid.pdf", ContentType: "application/pdf", Content: []byte("content")}, {Filename: "invalid\r\n.pdf", ContentType: "application/pdf"}}, expected: "email attachment filename cannot contain line breaks"},
	}

	for _, testCase := range testCases {
		s.Run(testCase.name, func() {
			err := s.sender.SendEmailWithAttachments(s.ctx, email.EmailWithAttachments{Email: email.Email{TextBody: &body}, Attachments: testCase.attachments})
			s.EqualError(err, testCase.expected)
		})
	}
}

func (s *sesSenderTestSuite) TestSendEmailNoBodyProvided() {
	s.EqualError(s.sender.SendEmail(s.ctx, email.Email{}), "email body cannot be empty")
}

func (s *sesSenderTestSuite) TestSendEmailInvalidSubject() {
	body := "This is a plain text email."
	s.EqualError(s.sender.SendEmail(s.ctx, email.Email{Subject: "Invoice\r\nBcc: attacker@example.com", TextBody: &body}), "email subject cannot contain line breaks")
}

func (s *sesSenderTestSuite) TestSendEmailErrorFromSES() {
	recipients := []string{"recipient@example.com"}
	subject := "Test Error Handling"
	body := "This email should trigger an error."
	expectedInput := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String("sender@example.com"),
		Destination:      &types.Destination{ToAddresses: recipients},
		Content: &types.EmailContent{Simple: &types.Message{
			Subject: &types.Content{Data: aws.String(subject), Charset: aws.String("UTF-8")},
			Body:    &types.Body{Text: &types.Content{Data: aws.String(body), Charset: aws.String("UTF-8")}},
		}},
	}
	s.client.EXPECT().SendEmail(matcher.Context, expectedInput).Return(nil, errors.New("error"))

	s.Error(s.sender.SendEmail(s.ctx, email.Email{Recipients: recipients, Subject: subject, TextBody: &body}))
}
