package email_test

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/email"
	emailMocks "github.com/justtrackio/gosoline/pkg/email/mocks"
	"github.com/justtrackio/gosoline/pkg/mdl"
	uuidMocks "github.com/justtrackio/gosoline/pkg/uuid/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type senderSmtpTestSuite struct {
	suite.Suite

	client *emailMocks.SmtpClient
	uuid   *uuidMocks.Uuid
	clock  clock.Clock
	from   string

	sender email.SenderWithAttachments
}

func TestRunSenderSmtpTestSuite(t *testing.T) {
	suite.Run(t, &senderSmtpTestSuite{})
}

func (s *senderSmtpTestSuite) SetupTest() {
	s.client = emailMocks.NewSmtpClient(s.T())
	s.uuid = uuidMocks.NewUuid(s.T())
	s.clock = clock.NewFakeClockAt(time.Date(2026, time.August, 21, 10, 30, 0, 0, time.UTC))
	s.from = "test@gosoline.com"

	clientFactory := func() (email.SmtpClient, error) {
		return s.client, nil
	}

	s.sender = email.NewSmtpSenderWithInterfaces(clientFactory, s.uuid, s.clock, s.from)
}

func (s *senderSmtpTestSuite) TestSendEmail_Html() {
	email := email.Email{
		Recipients: []string{"foo@bar.com"},
		Subject:    "Test Email",
		HtmlBody:   mdl.Box("<html><p><b>Hello!</b> We're sending you a test email.<p></html>"),
	}

	s.uuid.EXPECT().NewV4().Return("gosoMail")

	expectedBody := `Date: Fri, 21 Aug 2026 10:30:00 +0000
From: test@gosoline.com
To: foo@bar.com
Subject: Test Email
MIME-Version: 1.0
Content-Type: multipart/alternative; boundary=gosoMail

--gosoMail
Content-Disposition: inline
Content-Transfer-Encoding: quoted-printable
Content-Type: text/html; charset="utf-8"

<html><p><b>Hello!</b> We're sending you a test email.<p></html>

--gosoMail--
`

	expectedBody = strings.ReplaceAll(expectedBody, "\n", "\r\n")

	s.client.EXPECT().SendMail(s.from, []string{"foo@bar.com"}, mock.MatchedBy(func(val any) bool {
		_, ok := val.(io.Reader)

		return ok
	})).Run(func(_ string, _ []string, r io.Reader) {
		bytes, err := io.ReadAll(r)
		s.NoError(err)

		body := string(bytes)
		s.Equal(expectedBody, body)
	}).Return(nil)

	err := s.sender.SendEmail(s.T().Context(), email)
	s.NoError(err)
}

func (s *senderSmtpTestSuite) TestSendEmail_Text() {
	email := email.Email{
		Recipients: []string{"foo@bar.com"},
		Subject:    "Test Email",
		TextBody:   mdl.Box("Hello! We're sending you a test email."),
	}

	s.uuid.EXPECT().NewV4().Return("gosoMail")

	expectedBody := `Date: Fri, 21 Aug 2026 10:30:00 +0000
From: test@gosoline.com
To: foo@bar.com
Subject: Test Email
MIME-Version: 1.0
Content-Type: multipart/alternative; boundary=gosoMail

--gosoMail
Content-Disposition: inline
Content-Transfer-Encoding: quoted-printable
Content-Type: text/plain; charset="utf-8"

Hello! We're sending you a test email.

--gosoMail--
`

	expectedBody = strings.ReplaceAll(expectedBody, "\n", "\r\n")

	s.client.EXPECT().SendMail(s.from, []string{"foo@bar.com"}, mock.MatchedBy(func(val any) bool {
		_, ok := val.(io.Reader)

		return ok
	})).Run(func(_ string, _ []string, r io.Reader) {
		bytes, err := io.ReadAll(r)
		s.NoError(err)

		body := string(bytes)
		s.Equal(expectedBody, body)
	}).Return(nil)

	err := s.sender.SendEmail(s.T().Context(), email)
	s.NoError(err)
}

func (s *senderSmtpTestSuite) TestSendEmail_MultiFormat() {
	email := email.Email{
		Recipients: []string{"foo@bar.com"},
		Subject:    "Test Email",
		TextBody:   mdl.Box("Hello! We're sending you a test email."),
		HtmlBody:   mdl.Box("<html><p><b>Hello!</b> We're sending you a test email.<p></html>"),
	}

	s.uuid.EXPECT().NewV4().Return("gosoMail")

	expectedBody := `Date: Fri, 21 Aug 2026 10:30:00 +0000
From: test@gosoline.com
To: foo@bar.com
Subject: Test Email
MIME-Version: 1.0
Content-Type: multipart/alternative; boundary=gosoMail

--gosoMail
Content-Disposition: inline
Content-Transfer-Encoding: quoted-printable
Content-Type: text/plain; charset="utf-8"

Hello! We're sending you a test email.

--gosoMail
Content-Disposition: inline
Content-Transfer-Encoding: quoted-printable
Content-Type: text/html; charset="utf-8"

<html><p><b>Hello!</b> We're sending you a test email.<p></html>

--gosoMail--
`

	expectedBody = strings.ReplaceAll(expectedBody, "\n", "\r\n")

	s.client.EXPECT().SendMail(s.from, []string{"foo@bar.com"}, mock.MatchedBy(func(val any) bool {
		_, ok := val.(io.Reader)

		return ok
	})).Run(func(_ string, _ []string, r io.Reader) {
		bytes, err := io.ReadAll(r)
		s.NoError(err)

		body := string(bytes)
		s.Equal(expectedBody, body)
	}).Return(nil)

	err := s.sender.SendEmail(s.T().Context(), email)
	s.NoError(err)
}

func (s *senderSmtpTestSuite) TestSendEmailWithAttachments() {
	recipients := []string{"Foo Bar <foo@bar.com>"}
	subject := "Your résumé"
	textBody := "Your invoice is attached."
	htmlBody := "<p>Your invoice is attached.</p>"
	attachment := email.Attachment{
		Filename:    "invoice.pdf",
		ContentType: "application/pdf",
		Content:     []byte("%PDF-1.7\ninvoice"),
	}
	from := "Test Sender <test@gosoline.com>"
	sender := email.NewSmtpSenderWithInterfaces(func() (email.SmtpClient, error) {
		return s.client, nil
	}, s.uuid, s.clock, from)

	s.uuid.EXPECT().NewV4().Return("mixedBoundary").Once()
	s.uuid.EXPECT().NewV4().Return("alternativeBoundary").Once()
	s.client.EXPECT().SendMail("test@gosoline.com", []string{"foo@bar.com"}, mock.MatchedBy(func(val any) bool {
		_, ok := val.(io.Reader)

		return ok
	})).Run(func(_ string, _ []string, reader io.Reader) {
		assertAttachmentsMessage(s.T(), reader, from, recipients, subject, textBody, htmlBody, []email.Attachment{attachment})
	}).Return(nil)

	err := sender.SendEmailWithAttachments(s.T().Context(), email.EmailWithAttachments{
		Email: email.Email{
			Recipients: recipients,
			Subject:    subject,
			TextBody:   &textBody,
			HtmlBody:   &htmlBody,
		},
		Attachments: []email.Attachment{attachment},
	})
	s.NoError(err)
}

func (s *senderSmtpTestSuite) TestSendEmailWithAttachmentsFromSuiteSender() {
	textBody := "Your invoice is attached."
	attachment := email.Attachment{
		Filename:    "invoice.pdf",
		ContentType: "application/pdf",
		Content:     []byte("content"),
	}

	s.uuid.EXPECT().NewV4().Return("mixedBoundary").Once()
	s.uuid.EXPECT().NewV4().Return("alternativeBoundary").Once()
	s.client.EXPECT().SendMail(s.from, []string{"recipient@example.com"}, mock.AnythingOfType("*bytes.Reader")).Return(nil)

	err := s.sender.SendEmailWithAttachments(s.T().Context(), email.EmailWithAttachments{
		Email: email.Email{
			Recipients: []string{"recipient@example.com"},
			TextBody:   &textBody,
		},
		Attachments: []email.Attachment{attachment},
	})
	s.NoError(err)
}

func (s *senderSmtpTestSuite) TestSendEmail_NoBodyProvided() {
	err := s.sender.SendEmail(s.T().Context(), email.Email{})

	s.Error(err)
	s.EqualError(err, "email body cannot be empty")
}

func (s *senderSmtpTestSuite) TestSendEmail_InvalidEmailDoesNotDialSMTP() {
	body := "This is a plain text email."
	attachment := email.Attachment{
		Filename:    "invoice.pdf",
		ContentType: "application/pdf",
		Content:     []byte("content"),
	}

	testCases := []struct {
		name     string
		from     string
		email    email.Email
		message  *email.EmailWithAttachments
		expected string
	}{
		{
			name: "invalid subject",
			from: s.from,
			email: email.Email{
				Subject:  "Invoice\r\nBcc: attacker@example.com",
				TextBody: &body,
			},
			expected: "email subject cannot contain line breaks",
		},
		{
			name: "invalid sender",
			from: "invalid sender",
			email: email.Email{
				Recipients: []string{"recipient@example.com"},
				TextBody:   &body,
			},
			expected: "could not parse email envelope: format email sender:",
		},
		{
			name: "invalid attachment",
			from: s.from,
			message: &email.EmailWithAttachments{
				Email: email.Email{Recipients: []string{"recipient@example.com"}, TextBody: &body},
				Attachments: []email.Attachment{{
					ContentType: attachment.ContentType,
					Content:     attachment.Content,
				}},
			},
			expected: "email attachment filename cannot be empty",
		},
	}

	for _, testCase := range testCases {
		s.Run(testCase.name, func() {
			called := false
			sender := email.NewSmtpSenderWithInterfaces(func() (email.SmtpClient, error) {
				called = true

				return s.client, nil
			}, s.uuid, s.clock, testCase.from)

			var err error
			if testCase.message == nil {
				err = sender.SendEmail(s.T().Context(), testCase.email)
			} else {
				err = sender.SendEmailWithAttachments(s.T().Context(), *testCase.message)
			}
			s.ErrorContains(err, testCase.expected)
			s.False(called)
		})
	}
}
