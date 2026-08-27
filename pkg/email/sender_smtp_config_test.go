package email

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	uuidMocks "github.com/justtrackio/gosoline/pkg/uuid/mocks"
	"github.com/stretchr/testify/require"
)

func TestSmtpSenderConfiguredEncodedMessageSizeRejectsBeforeDialing(t *testing.T) {
	t.Parallel()

	config := cfg.New(map[string]any{
		"email": map[string]any{
			"test": map[string]any{
				"server":                   "smtp.example.com:25",
				"from_address":             "sender@example.com",
				"max_encoded_message_size": 1,
			},
		},
	})
	uuid := uuidMocks.NewUuid(t)
	uuid.EXPECT().NewV4().Return("mixedBoundary").Once()
	uuid.EXPECT().NewV4().Return("alternativeBoundary").Once()

	dials := 0
	sender, err := newSmtpSenderFromConfig(config, "test", func(server string) (SmtpClient, error) {
		require.Equal(t, "smtp.example.com:25", server)
		dials++

		return nil, nil
	}, uuid, clock.NewFakeClock())
	require.NoError(t, err)
	require.Equal(t, 1, dials)
	dials = 0

	body := "Attachment body"
	err = sender.SendEmailWithAttachments(t.Context(), EmailWithAttachments{
		Email: Email{
			Recipients: []string{"recipient@example.com"},
			TextBody:   &body,
		},
		Attachments: []Attachment{{
			Filename:    "attachment.txt",
			ContentType: "text/plain",
			Content:     []byte("attachment content"),
		}},
	})

	require.ErrorContains(t, err, "encoded email message size exceeds configured limit of 1 bytes")
	require.Zero(t, dials)
}
