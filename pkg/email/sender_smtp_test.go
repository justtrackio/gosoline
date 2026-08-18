package email_test

import (
	"io"
	"strings"
	"testing"

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
	from   string

	sender email.Sender
}

func TestRunSenderSmtpTestSuite(t *testing.T) {
	suite.Run(t, &senderSmtpTestSuite{})
}

func (s *senderSmtpTestSuite) SetupTest() {
	s.client = emailMocks.NewSmtpClient(s.T())
	s.uuid = uuidMocks.NewUuid(s.T())
	s.from = "test@gosoline.com"

	clientFactory := func() (email.SmtpClient, error) {
		return s.client, nil
	}

	s.sender = email.NewSmtpSenderWithInterfaces(clientFactory, s.uuid, s.from)
}

func (s *senderSmtpTestSuite) TestSendEmail_Html() {
	email := email.Email{
		Recipients: []string{"foo@bar.com"},
		Subject:    "Test Email",
		HtmlBody:   mdl.Box("<html><p><b>Hello!</b> We're sending you a test email.<p></html>"),
	}

	s.uuid.EXPECT().NewV4().Return("gosoMail")

	expectedBody := `Subject: Test Email
MIME-Version: 1.0
Content-Type: multipart/alternative; boundary="gosoMail"

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

	expectedBody := `Subject: Test Email
MIME-Version: 1.0
Content-Type: multipart/alternative; boundary="gosoMail"

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

	expectedBody := `Subject: Test Email
MIME-Version: 1.0
Content-Type: multipart/alternative; boundary="gosoMail"

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

func (s *senderSmtpTestSuite) TestSendEmail_WithAttachment() {
	recipients := []string{"foo@bar.com"}
	subject := "Your résumé"
	textBody := "Your invoice is attached."
	htmlBody := "<p>Your invoice is attached.</p>"
	attachment := &email.Attachment{
		Filename:    "invoice.pdf",
		ContentType: "application/pdf",
		Content:     []byte("%PDF-1.7\ninvoice"),
	}

	s.uuid.EXPECT().NewV4().Return("mixedBoundary").Once()
	s.uuid.EXPECT().NewV4().Return("alternativeBoundary").Once()
	s.client.EXPECT().SendMail(s.from, recipients, mock.MatchedBy(func(val any) bool {
		_, ok := val.(io.Reader)

		return ok
	})).Run(func(_ string, _ []string, reader io.Reader) {
		assertAttachmentMessage(s.T(), reader, s.from, recipients, subject, textBody, htmlBody, attachment)
	}).Return(nil)

	err := s.sender.SendEmail(s.T().Context(), email.Email{
		Recipients: recipients,
		Subject:    subject,
		TextBody:   &textBody,
		HtmlBody:   &htmlBody,
		Attachment: attachment,
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
	attachment := &email.Attachment{
		Filename:    "invoice.pdf",
		ContentType: "application/pdf",
		Content:     []byte("content"),
	}

	testCases := []struct {
		name     string
		from     string
		email    email.Email
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
			name: "invalid sender for attachment",
			from: "invalid sender",
			email: email.Email{
				Recipients: []string{"recipient@example.com"},
				TextBody:   &body,
				Attachment: attachment,
			},
			expected: "could not compile email body: format email sender:",
		},
	}

	for _, testCase := range testCases {
		s.Run(testCase.name, func() {
			called := false
			sender := email.NewSmtpSenderWithInterfaces(func() (email.SmtpClient, error) {
				called = true

				return s.client, nil
			}, s.uuid, testCase.from)

			err := sender.SendEmail(s.T().Context(), testCase.email)
			s.ErrorContains(err, testCase.expected)
			s.False(called)
		})
	}
}
