package email

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCompileAttachmentMIME(t *testing.T) {
	text := "Plain text body"
	html := "<p>HTML body</p>"
	attachmentContent := bytes.Repeat([]byte("a"), 58)
	boundaries := []string{"mixed-boundary", "alternative-boundary"}

	rawMessage, err := compileAttachmentMIME(
		Email{
			Recipients: []string{"First Recipient <first@example.com>", "second@example.com"},
			Subject:    "Your résumé",
			TextBody:   &text,
			HtmlBody:   &html,
			Attachment: &Attachment{
				Filename:    "résumé.pdf",
				ContentType: "application/pdf; version=1",
				Content:     attachmentContent,
			},
		},
		"Sender <sender@example.com>",
		func() string {
			boundary := boundaries[0]
			boundaries = boundaries[1:]

			return boundary
		},
	)
	require.NoError(t, err)
	require.Empty(t, boundaries)

	message, err := mail.ReadMessage(bytes.NewReader(rawMessage))
	require.NoError(t, err)

	sender, err := mail.ParseAddress(message.Header.Get("From"))
	require.NoError(t, err)
	require.Equal(t, "sender@example.com", sender.Address)

	recipients, err := mail.ParseAddressList(message.Header.Get("To"))
	require.NoError(t, err)
	require.Len(t, recipients, 2)
	require.Equal(t, []string{"first@example.com", "second@example.com"}, []string{recipients[0].Address, recipients[1].Address})

	subject, err := new(mime.WordDecoder).DecodeHeader(message.Header.Get("Subject"))
	require.NoError(t, err)
	require.Equal(t, "Your résumé", subject)
	require.Equal(t, "1.0", message.Header.Get("MIME-Version"))
	_, err = time.Parse(time.RFC1123Z, message.Header.Get("Date"))
	require.NoError(t, err)

	messageContentType, messageParams, err := mime.ParseMediaType(message.Header.Get("Content-Type"))
	require.NoError(t, err)
	require.Equal(t, "multipart/mixed", messageContentType)
	require.Equal(t, "mixed-boundary", messageParams["boundary"])

	mixedReader := multipart.NewReader(message.Body, messageParams["boundary"])
	alternativePart, err := mixedReader.NextPart()
	require.NoError(t, err)

	alternativeContentType, alternativeParams, err := mime.ParseMediaType(alternativePart.Header.Get("Content-Type"))
	require.NoError(t, err)
	require.Equal(t, "multipart/alternative", alternativeContentType)
	require.Equal(t, "alternative-boundary", alternativeParams["boundary"])

	alternativeReader := multipart.NewReader(alternativePart, alternativeParams["boundary"])
	textPart, err := alternativeReader.NextPart()
	require.NoError(t, err)
	require.Equal(t, "text/plain", mediaType(t, textPart))
	require.Equal(t, text+mimeLineBreak, decodedQuotedPrintable(t, textPart))

	htmlPart, err := alternativeReader.NextPart()
	require.NoError(t, err)
	require.Equal(t, "text/html", mediaType(t, htmlPart))
	require.Equal(t, html+mimeLineBreak, decodedQuotedPrintable(t, htmlPart))

	_, err = alternativeReader.NextPart()
	require.ErrorIs(t, err, io.EOF)

	attachmentPart, err := mixedReader.NextPart()
	require.NoError(t, err)
	require.Equal(t, "base64", attachmentPart.Header.Get("Content-Transfer-Encoding"))

	attachmentContentType, attachmentParams, err := mime.ParseMediaType(attachmentPart.Header.Get("Content-Type"))
	require.NoError(t, err)
	require.Equal(t, "application/pdf", attachmentContentType)
	require.Equal(t, "résumé.pdf", attachmentParams["name"])
	require.Equal(t, "1", attachmentParams["version"])

	disposition, dispositionParams, err := mime.ParseMediaType(attachmentPart.Header.Get("Content-Disposition"))
	require.NoError(t, err)
	require.Equal(t, "attachment", disposition)
	require.Equal(t, "résumé.pdf", dispositionParams["filename"])

	require.Equal(t, attachmentContent, decodeMIMEBase64(t, attachmentPart))

	_, err = mixedReader.NextPart()
	require.ErrorIs(t, err, io.EOF)
}

func TestCompileAttachmentMIMEValidation(t *testing.T) {
	text := "body"
	email := Email{
		Recipients: []string{"recipient@example.com"},
		TextBody:   &text,
		Attachment: &Attachment{
			Filename:    "document.pdf",
			ContentType: "application/pdf",
			Content:     []byte("content"),
		},
	}

	_, err := compileAttachmentMIME(email, "invalid sender", generatedMIMEBoundary)
	require.ErrorContains(t, err, "format email sender:")

	email.Recipients = nil
	_, err = compileAttachmentMIME(email, "sender@example.com", generatedMIMEBoundary)
	require.EqualError(t, err, "format email recipients: recipient list is empty")

	email.Recipients = []string{"invalid recipient"}
	_, err = compileAttachmentMIME(email, "sender@example.com", generatedMIMEBoundary)
	require.ErrorContains(t, err, "format email recipients:")

	email.Recipients = []string{"recipient@example.com"}
	email.Attachment.ContentType = "application/pdf; ="
	_, err = compileAttachmentMIME(email, "sender@example.com", generatedMIMEBoundary)
	require.ErrorContains(t, err, "parse email attachment content type:")
}

func mediaType(t *testing.T, part *multipart.Part) string {
	t.Helper()

	contentType, _, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
	require.NoError(t, err)

	return contentType
}

func decodedQuotedPrintable(t *testing.T, part *multipart.Part) string {
	t.Helper()

	body, err := io.ReadAll(quotedprintable.NewReader(part))
	require.NoError(t, err)

	return string(body)
}
