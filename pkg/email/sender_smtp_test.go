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
Content-Type: multipart/alternative; boundary="gosoMail"

--gosoMail
Content-Disposition: inline
Content-Transfer-Encoding: quoted-printable
Content-Type: text/html; charset="utf-8"

<html><p><b>Hello!</b> We're sending you a test email.<p></html>

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

func (s *senderSmtpTestSuite) TestSendEmail_NoBodyProvided() {
	err := s.sender.SendEmail(s.T().Context(), email.Email{})

	s.Error(err)
	s.EqualError(err, "email body cannot be empty")
}
