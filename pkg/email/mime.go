package email

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"strings"
	"time"
)

const mimeLineBreak = "\r\n"

func compileAttachmentMIME(email Email, fromAddress string, nextBoundary func() string) ([]byte, error) {
	from, err := formatMailbox(fromAddress)
	if err != nil {
		return nil, fmt.Errorf("format email sender: %w", err)
	}
	to, err := formatMailboxList(email.Recipients)
	if err != nil {
		return nil, fmt.Errorf("format email recipients: %w", err)
	}

	mixedBoundary := nextBoundary()
	alternativeBoundary := nextBoundary()
	body := &bytes.Buffer{}
	if err := writeRawMessageHeaders(body, from, to, email.Subject, "multipart/mixed", mixedBoundary, time.Now().UTC()); err != nil {
		return nil, err
	}

	mixedWriter := multipart.NewWriter(body)
	if err := mixedWriter.SetBoundary(mixedBoundary); err != nil {
		return nil, fmt.Errorf("could not write email boundary: %w", err)
	}
	if err := writeAlternativePart(mixedWriter, alternativeBoundary, email.TextBody, email.HtmlBody); err != nil {
		return nil, err
	}
	if err := writeAttachmentPart(mixedWriter, email.Attachment); err != nil {
		return nil, err
	}
	if err := mixedWriter.Close(); err != nil {
		return nil, fmt.Errorf("could not close multipart writer: %w", err)
	}

	return body.Bytes(), nil
}

func formatMailbox(value string) (string, error) {
	mailbox, err := mail.ParseAddress(value)
	if err != nil {
		return "", err
	}

	return mailbox.String(), nil
}

func formatMailboxList(recipients []string) (string, error) {
	if len(recipients) == 0 {
		return "", fmt.Errorf("recipient list is empty")
	}

	mailboxes := make([]string, 0, len(recipients))
	for _, recipient := range recipients {
		mailbox, err := formatMailbox(recipient)
		if err != nil {
			return "", err
		}
		mailboxes = append(mailboxes, mailbox)
	}

	return strings.Join(mailboxes, ", "), nil
}

func writeAlternativePart(mixedWriter *multipart.Writer, alternativeBoundary string, text *string, html *string) error {
	alternativeHeader := textproto.MIMEHeader{}
	alternativeHeader.Set("Content-Type", mime.FormatMediaType("multipart/alternative", map[string]string{"boundary": alternativeBoundary}))
	alternativePart, err := mixedWriter.CreatePart(alternativeHeader)
	if err != nil {
		return fmt.Errorf("could not create email body part: %w", err)
	}

	alternativeWriter := multipart.NewWriter(alternativePart)
	if err := alternativeWriter.SetBoundary(alternativeBoundary); err != nil {
		return fmt.Errorf("could not write alternative email boundary: %w", err)
	}
	if err := writeBodyParts(alternativeWriter, text, html); err != nil {
		return err
	}
	if err := alternativeWriter.Close(); err != nil {
		return fmt.Errorf("could not close alternative multipart writer: %w", err)
	}

	return nil
}

func writeAttachmentPart(mixedWriter *multipart.Writer, attachment *Attachment) error {
	mediaType, params, err := mime.ParseMediaType(attachment.ContentType)
	if err != nil {
		return fmt.Errorf("parse email attachment content type: %w", err)
	}
	params["name"] = attachment.Filename

	attachmentHeader := textproto.MIMEHeader{}
	attachmentHeader.Set("Content-Type", mime.FormatMediaType(mediaType, params))
	attachmentHeader.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": attachment.Filename}))
	attachmentHeader.Set("Content-Transfer-Encoding", "base64")
	attachmentPart, err := mixedWriter.CreatePart(attachmentHeader)
	if err != nil {
		return fmt.Errorf("could not create email attachment part: %w", err)
	}
	if err := writeMIMEBase64(attachmentPart, attachment.Content); err != nil {
		return fmt.Errorf("could not encode email attachment: %w", err)
	}

	return nil
}

func generatedMIMEBoundary() string {
	return multipart.NewWriter(io.Discard).Boundary()
}

func writeRawMessageHeaders(body *bytes.Buffer, from string, to string, subject string, contentType string, boundary string, now time.Time) error {
	headers := []string{
		"Date: " + now.Format(time.RFC1123Z) + mimeLineBreak,
		"From: " + from + mimeLineBreak,
		"To: " + to + mimeLineBreak,
		"Subject: " + encodeHeader(subject) + mimeLineBreak,
		"MIME-Version: 1.0" + mimeLineBreak,
		fmt.Sprintf("Content-Type: %s; boundary=%q", contentType, boundary) + mimeLineBreak,
		mimeLineBreak,
	}

	return writeHeaders(body, headers)
}

func writeMessageHeaders(body *bytes.Buffer, subject string, contentType string, boundary string) error {
	headers := []string{
		"Subject: " + subject + mimeLineBreak,
		"MIME-Version: 1.0" + mimeLineBreak,
		fmt.Sprintf("Content-Type: %s; boundary=%q", contentType, boundary) + mimeLineBreak,
		mimeLineBreak,
	}

	return writeHeaders(body, headers)
}

func writeHeaders(body *bytes.Buffer, headers []string) error {
	for _, header := range headers {
		if _, err := body.WriteString(header); err != nil {
			return fmt.Errorf("could not write email header: %w", err)
		}
	}

	return nil
}

func writeBodyParts(writer *multipart.Writer, text *string, html *string) error {
	if text != nil {
		textBody, err := writer.CreatePart(mimeHeader("text/plain"))
		if err != nil {
			return fmt.Errorf("could not create email header part: %w", err)
		}

		textBytes, err := encodeQuotedPrintable(*text + mimeLineBreak)
		if err != nil {
			return fmt.Errorf("could not encode text body: %w", err)
		}
		if _, err := textBody.Write(textBytes); err != nil {
			return fmt.Errorf("could not write text body: %w", err)
		}
	}

	if html != nil {
		htmlBody, err := writer.CreatePart(mimeHeader("text/html"))
		if err != nil {
			return fmt.Errorf("could not create email header part: %w", err)
		}

		htmlBytes, err := encodeQuotedPrintable(*html + mimeLineBreak)
		if err != nil {
			return fmt.Errorf("could not encode html body: %w", err)
		}
		if _, err := htmlBody.Write(htmlBytes); err != nil {
			return fmt.Errorf("could not write html body: %w", err)
		}
	}

	return nil
}

func encodeHeader(data string) string {
	return mime.QEncoding.Encode("UTF-8", data)
}

func encodeQuotedPrintable(data string) ([]byte, error) {
	quoted := &bytes.Buffer{}
	writer := quotedprintable.NewWriter(quoted)
	if _, err := writer.Write([]byte(data)); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	return quoted.Bytes(), nil
}

func mimeHeader(contentType string) textproto.MIMEHeader {
	return textproto.MIMEHeader{
		"Content-Type":              []string{contentType + "; charset=\"utf-8\""},
		"Content-Transfer-Encoding": []string{"quoted-printable"},
		"Content-Disposition":       []string{"inline"},
	}
}
