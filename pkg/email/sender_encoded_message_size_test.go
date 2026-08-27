package email_test

import (
	"strings"
	"testing"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/email"
	uuidMocks "github.com/justtrackio/gosoline/pkg/uuid/mocks"
	"github.com/stretchr/testify/require"
)

func TestSmtpSenderWithAttachments_EncodedMessageExceedsDefaultLimitDoesNotDialSMTP(t *testing.T) {
	t.Parallel()

	uuid := uuidMocks.NewUuid(t)
	uuid.EXPECT().NewV4().Return("mixedBoundary").Once()
	uuid.EXPECT().NewV4().Return("alternativeBoundary").Once()

	dialed := false
	sender := email.NewSmtpSenderWithInterfaces(func() (email.SmtpClient, error) {
		dialed = true

		return nil, nil
	}, uuid, clock.NewFakeClock(), "sender@example.com")

	body := "The combined attachments exceed the encoded message size limit."
	attachmentContent := []byte(strings.Repeat("a", 4*1024*1024))
	err := sender.SendEmailWithAttachments(t.Context(), email.EmailWithAttachments{
		Email: email.Email{
			Recipients: []string{"recipient@example.com"},
			TextBody:   &body,
		},
		Attachments: []email.Attachment{
			{Filename: "first.bin", ContentType: "application/octet-stream", Content: attachmentContent},
			{Filename: "second.bin", ContentType: "application/octet-stream", Content: attachmentContent},
		},
	})

	require.Error(t, err)
	require.Regexp(t, `encoded email message size exceeds configured limit of 10485760 bytes`, err.Error())
	require.False(t, dialed)
}
