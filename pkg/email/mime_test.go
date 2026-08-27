package email

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var testMessageDate = time.Date(2026, time.August, 21, 10, 30, 0, 0, time.UTC)

func compileMIMEForTest(email EmailWithAttachments, fromAddress string, nextBoundary func() string) ([]byte, error) {
	envelope, err := parseEmailEnvelope(fromAddress, email.Recipients)
	if err != nil {
		return nil, err
	}

	body := &bytes.Buffer{}
	if err := compileMIME(email, envelope, nextBoundary, testMessageDate, body); err != nil {
		return nil, err
	}

	return body.Bytes(), nil
}

func TestCompileAttachmentsMIME(t *testing.T) {
	text := "Plain text body for a résumé = 1"
	html := "<p>HTML body for a résumé = 1</p>"
	attachments := []Attachment{
		{Filename: "résumé.pdf", ContentType: "application/pdf; version=1", Content: bytes.Repeat([]byte("a"), 58)},
		{Filename: "details.txt", ContentType: "text/plain", Content: []byte("second attachment")},
	}
	boundaries := []string{"mixed-boundary", "alternative-boundary"}

	rawMessage, err := compileMIMEForTest(
		EmailWithAttachments{
			Email: Email{
				Recipients: []string{"First Recipient <first@example.com>, second@example.com"},
				Subject:    "Your résumé",
				TextBody:   &text,
				HtmlBody:   &html,
			},
			Attachments: attachments,
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
	require.Equal(t, "Sender", sender.Name)
	recipients, err := mail.ParseAddressList(message.Header.Get("To"))
	require.NoError(t, err)
	require.Len(t, recipients, 2)
	require.Equal(t, []string{"first@example.com", "second@example.com"}, []string{recipients[0].Address, recipients[1].Address})
	require.Equal(t, "First Recipient", recipients[0].Name)
	subject, err := new(mime.WordDecoder).DecodeHeader(message.Header.Get("Subject"))
	require.NoError(t, err)
	require.Equal(t, "Your résumé", subject)
	require.Equal(t, "1.0", message.Header.Get("MIME-Version"))
	date, err := time.Parse(time.RFC1123Z, message.Header.Get("Date"))
	require.NoError(t, err)
	require.True(t, testMessageDate.Equal(date))

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
	require.Equal(t, text+mimeLineBreak, partBody(t, textPart))

	htmlPart, err := alternativeReader.NextPart()
	require.NoError(t, err)
	require.Equal(t, "text/html", mediaType(t, htmlPart))
	require.Equal(t, html+mimeLineBreak, partBody(t, htmlPart))

	_, err = alternativeReader.NextPart()
	require.ErrorIs(t, err, io.EOF)

	for _, attachment := range attachments {
		assertMIMEAttachmentPart(t, mixedReader, attachment)
	}
	_, err = mixedReader.NextPart()
	require.ErrorIs(t, err, io.EOF)
}

func TestCompileMIMEWithoutAttachments(t *testing.T) {
	text := "Plain text body for a résumé = 1"
	html := "<p>HTML body for a résumé = 1</p>"

	rawMessage, err := compileMIMEForTest(
		EmailWithAttachments{Email: Email{
			Recipients: []string{"First Recipient <first@example.com>", "second@example.com"},
			Subject:    "Your résumé",
			TextBody:   &text,
			HtmlBody:   &html,
		}},
		"Sender <sender@example.com>",
		func() string { return "alternative-boundary" },
	)
	require.NoError(t, err)

	message, err := mail.ReadMessage(bytes.NewReader(rawMessage))
	require.NoError(t, err)

	sender, err := mail.ParseAddress(message.Header.Get("From"))
	require.NoError(t, err)
	require.Equal(t, "Sender", sender.Name)
	require.Equal(t, "sender@example.com", sender.Address)

	recipients, err := mail.ParseAddressList(message.Header.Get("To"))
	require.NoError(t, err)
	require.Len(t, recipients, 2)
	require.Equal(t, "First Recipient", recipients[0].Name)
	require.Equal(t, []string{"first@example.com", "second@example.com"}, []string{recipients[0].Address, recipients[1].Address})

	date, err := time.Parse(time.RFC1123Z, message.Header.Get("Date"))
	require.NoError(t, err)
	require.True(t, testMessageDate.Equal(date))
	require.Equal(t, "1.0", message.Header.Get("MIME-Version"))

	subject, err := new(mime.WordDecoder).DecodeHeader(message.Header.Get("Subject"))
	require.NoError(t, err)
	require.Equal(t, "Your résumé", subject)

	contentType, params, err := mime.ParseMediaType(message.Header.Get("Content-Type"))
	require.NoError(t, err)
	require.Equal(t, "multipart/alternative", contentType)
	require.Equal(t, "alternative-boundary", params["boundary"])

	reader := multipart.NewReader(message.Body, params["boundary"])
	textPart, err := reader.NextPart()
	require.NoError(t, err)
	require.Equal(t, "text/plain", mediaType(t, textPart))
	require.Equal(t, text+mimeLineBreak, partBody(t, textPart))

	htmlPart, err := reader.NextPart()
	require.NoError(t, err)
	require.Equal(t, "text/html", mediaType(t, htmlPart))
	require.Equal(t, html+mimeLineBreak, partBody(t, htmlPart))

	_, err = reader.NextPart()
	require.ErrorIs(t, err, io.EOF)
}

func TestFormatMailboxAddress(t *testing.T) {
	longAsciiName := strings.Repeat("Very Long Sender ", 4) + "Name"
	formatted := formatMailboxAddress(&mail.Address{Name: longAsciiName, Address: "sender@example.com"})
	require.Greater(t, len(formatted), recommendedHeaderLineLength)
	require.NotContains(t, formatted, "=?UTF-8?B?")

	folded, err := foldHeader("From", formatted)
	require.NoError(t, err)
	for _, line := range strings.Split(folded, mimeLineBreak) {
		require.LessOrEqual(t, len(line), maximumHeaderLineLength)
	}

	parsed, err := mail.ParseAddress(formatted)
	require.NoError(t, err)
	require.Equal(t, longAsciiName, parsed.Name)
	require.Equal(t, "sender@example.com", parsed.Address)

	longUnicodeName := strings.Repeat("Résumé Sender ", 6) + "Name"
	formatted = formatMailboxAddress(&mail.Address{Name: longUnicodeName, Address: "sender@example.com"})
	require.Contains(t, formatted, "=?UTF-8?B?")

	parsed, err = mail.ParseAddress(formatted)
	require.NoError(t, err)
	require.Equal(t, longUnicodeName, parsed.Name)
	require.Equal(t, "sender@example.com", parsed.Address)
}

func TestFormatLongMediaTypeParameters(t *testing.T) {
	name := strings.Repeat("é", 60)
	filename := strings.Repeat("a", 60)

	mediaType, params, err := mime.ParseMediaType(formatLongMediaTypeParameters("application/pdf", map[string]string{
		// the media type parameters of an attachment arrive lower cased, but the encoder must not depend on that
		"Name":     name,
		"filename": filename,
	}))
	require.NoError(t, err)
	require.Equal(t, "application/pdf", mediaType)
	require.Equal(t, name, params["name"])
	require.Equal(t, filename, params["filename"])
}

func TestCompileAttachmentsMIMEFoldsLongHeaders(t *testing.T) {
	text := "body"
	filename := strings.Repeat("résumé-", 180) + ".pdf"
	subject := strings.Repeat("résumé ", 220)
	description := strings.Repeat("a", 1_200)
	recipients := make([]string, 30)
	for i := range recipients {
		recipients[i] = "Recipient " + strings.Repeat("Name ", 4) + "<recipient-" + strings.Repeat("x", 15) + string(rune('a'+i%26)) + "@example.com>"
	}

	rawMessage, err := compileMIMEForTest(
		EmailWithAttachments{
			Email: Email{Recipients: recipients, Subject: subject, TextBody: &text},
			Attachments: []Attachment{
				{Filename: filename, ContentType: "application/pdf; description=" + description, Content: []byte("content")},
			},
		},
		"Sender <sender@example.com>",
		generatedMIMEBoundary,
	)
	require.NoError(t, err)

	for _, line := range bytes.Split(rawMessage, []byte(mimeLineBreak)) {
		require.LessOrEqual(t, len(line), maximumHeaderLineLength)
	}
	require.Contains(t, string(rawMessage), mimeLineBreak+" ")

	message, err := mail.ReadMessage(bytes.NewReader(rawMessage))
	require.NoError(t, err)
	parsedRecipients, err := mail.ParseAddressList(message.Header.Get("To"))
	require.NoError(t, err)
	require.Len(t, parsedRecipients, len(recipients))

	decodedSubject, err := new(mime.WordDecoder).DecodeHeader(message.Header.Get("Subject"))
	require.NoError(t, err)
	require.Equal(t, subject, decodedSubject)

	_, messageParams, err := mime.ParseMediaType(message.Header.Get("Content-Type"))
	require.NoError(t, err)
	mixedReader := multipart.NewReader(message.Body, messageParams["boundary"])
	_, err = mixedReader.NextPart()
	require.NoError(t, err)
	assertMIMEAttachmentPart(t, mixedReader, Attachment{Filename: filename, ContentType: "application/pdf; description=" + description, Content: []byte("content")})
}

func TestCompileAttachmentsMIMEValidation(t *testing.T) {
	text := "body"
	email := EmailWithAttachments{
		Email:       Email{Recipients: []string{"recipient@example.com"}, TextBody: &text},
		Attachments: []Attachment{{Filename: "document.pdf", ContentType: "application/pdf", Content: []byte("content")}, {Filename: "invalid.pdf", ContentType: "application/pdf; =", Content: []byte("content")}},
	}

	_, err := compileMIMEForTest(email, "invalid sender", generatedMIMEBoundary)
	require.ErrorContains(t, err, "format email sender:")

	email.Recipients = nil
	_, err = compileMIMEForTest(email, "sender@example.com", generatedMIMEBoundary)
	require.EqualError(t, err, "format email recipients: recipient list is empty")

	email.Recipients = []string{"invalid recipient"}
	_, err = compileMIMEForTest(email, "sender@example.com", generatedMIMEBoundary)
	require.ErrorContains(t, err, "format email recipients:")

	email.Recipients = []string{"recipient@example.com"}
	_, err = compileMIMEForTest(email, "sender@example.com", generatedMIMEBoundary)
	require.ErrorContains(t, err, "parse email attachment content type:")
}

func TestEmailWithAttachmentsValidation(t *testing.T) {
	text := "body"
	message := EmailWithAttachments{Email: Email{TextBody: &text}}

	require.EqualError(t, message.validate(), "email attachments cannot be empty")

	message.Attachments = []Attachment{
		{Filename: "valid.txt", ContentType: "text/plain", Content: []byte("valid")},
		{Filename: "empty.txt", ContentType: "text/plain"},
	}
	require.NoError(t, message.validate())
}

func assertMIMEAttachmentPart(t testing.TB, mixedReader *multipart.Reader, attachment Attachment) {
	t.Helper()

	attachmentPart, err := mixedReader.NextPart()
	require.NoError(t, err)
	require.Equal(t, "base64", attachmentPart.Header.Get("Content-Transfer-Encoding"))

	attachmentContentType, attachmentParams, err := mime.ParseMediaType(attachmentPart.Header.Get("Content-Type"))
	require.NoError(t, err)
	expectedAttachmentContentType, expectedAttachmentParams, err := mime.ParseMediaType(attachment.ContentType)
	require.NoError(t, err)
	require.Equal(t, expectedAttachmentContentType, attachmentContentType)
	require.Equal(t, attachment.Filename, attachmentParams["name"])
	for key, value := range expectedAttachmentParams {
		require.Equal(t, value, attachmentParams[key])
	}

	disposition, dispositionParams, err := mime.ParseMediaType(attachmentPart.Header.Get("Content-Disposition"))
	require.NoError(t, err)
	require.Equal(t, "attachment", disposition)
	require.Equal(t, attachment.Filename, dispositionParams["filename"])
	require.Equal(t, attachment.Content, decodeMIMEBase64(t, attachmentPart))
}

func mediaType(t *testing.T, part *multipart.Part) string {
	t.Helper()

	contentType, _, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
	require.NoError(t, err)

	return contentType
}

func partBody(t *testing.T, part *multipart.Part) string {
	t.Helper()

	body, err := io.ReadAll(part)
	require.NoError(t, err)

	return string(body)
}
