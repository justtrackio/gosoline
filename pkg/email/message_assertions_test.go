package email_test

import (
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"testing"

	"github.com/justtrackio/gosoline/pkg/email"
	"github.com/stretchr/testify/require"
)

func assertAttachmentsMessage(t testing.TB, reader io.Reader, from string, recipients []string, subject string, textBody string, htmlBody string, attachments []email.Attachment) {
	t.Helper()

	message, err := mail.ReadMessage(reader)
	require.NoError(t, err)

	sender, err := mail.ParseAddress(message.Header.Get("From"))
	require.NoError(t, err)
	expectedSender, err := mail.ParseAddress(from)
	require.NoError(t, err)
	require.Equal(t, expectedSender.Address, sender.Address)
	require.Equal(t, expectedSender.Name, sender.Name)

	recipientAddresses, err := mail.ParseAddressList(message.Header.Get("To"))
	require.NoError(t, err)
	require.Len(t, recipientAddresses, len(recipients))
	for index, recipient := range recipientAddresses {
		expectedRecipient, err := mail.ParseAddress(recipients[index])
		require.NoError(t, err)
		require.Equal(t, expectedRecipient.Address, recipient.Address)
		require.Equal(t, expectedRecipient.Name, recipient.Name)
	}

	require.Equal(t, "1.0", message.Header.Get("MIME-Version"))
	_, err = mail.ParseDate(message.Header.Get("Date"))
	require.NoError(t, err)

	decodedSubject, err := new(mime.WordDecoder).DecodeHeader(message.Header.Get("Subject"))
	require.NoError(t, err)
	require.Equal(t, subject, decodedSubject)

	contentType, params, err := mime.ParseMediaType(message.Header.Get("Content-Type"))
	require.NoError(t, err)
	require.Equal(t, "multipart/mixed", contentType)

	mixedReader := multipart.NewReader(message.Body, params["boundary"])
	alternativePart, err := mixedReader.NextPart()
	require.NoError(t, err)

	alternativeContentType, alternativeParams, err := mime.ParseMediaType(alternativePart.Header.Get("Content-Type"))
	require.NoError(t, err)
	require.Equal(t, "multipart/alternative", alternativeContentType)

	alternativeReader := multipart.NewReader(alternativePart, alternativeParams["boundary"])
	textPart, err := alternativeReader.NextPart()
	require.NoError(t, err)
	require.Equal(t, "text/plain", partMediaType(t, textPart))
	require.Equal(t, textBody+"\r\n", decodePart(t, textPart))

	htmlPart, err := alternativeReader.NextPart()
	require.NoError(t, err)
	require.Equal(t, "text/html", partMediaType(t, htmlPart))
	require.Equal(t, htmlBody+"\r\n", decodePart(t, htmlPart))

	_, err = alternativeReader.NextPart()
	require.ErrorIs(t, err, io.EOF)

	for _, attachment := range attachments {
		attachmentPart, err := mixedReader.NextPart()
		require.NoError(t, err)
		attachmentContentType, _, err := mime.ParseMediaType(attachmentPart.Header.Get("Content-Type"))
		require.NoError(t, err)
		expectedAttachmentContentType, _, err := mime.ParseMediaType(attachment.ContentType)
		require.NoError(t, err)
		require.Equal(t, expectedAttachmentContentType, attachmentContentType)

		disposition, dispositionParams, err := mime.ParseMediaType(attachmentPart.Header.Get("Content-Disposition"))
		require.NoError(t, err)
		require.Equal(t, "attachment", disposition)
		require.Equal(t, attachment.Filename, dispositionParams["filename"])

		content, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, attachmentPart))
		require.NoError(t, err)
		require.Equal(t, attachment.Content, content)
	}

	_, err = mixedReader.NextPart()
	require.ErrorIs(t, err, io.EOF)
}

func partMediaType(t testing.TB, part *multipart.Part) string {
	t.Helper()

	contentType, _, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
	require.NoError(t, err)

	return contentType
}

func decodePart(t testing.TB, part *multipart.Part) string {
	t.Helper()

	body, err := io.ReadAll(part)
	require.NoError(t, err)

	return string(body)
}
