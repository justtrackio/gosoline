package email

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"
	"time"
)

// compileMIME renders email as a MIME message. A message without attachments becomes a multipart/alternative message, a
// message with attachments a multipart/mixed message whose first part carries the multipart/alternative bodies. Both shapes
// get the same header block, so the presence of an attachment does not change how the message is addressed.
func compileMIME(email EmailWithAttachments, envelope emailEnvelope, nextBoundary func() string, now time.Time, body io.Writer) error {
	if len(email.Attachments) == 0 {
		return compileAlternativeMIME(email.Email, envelope, nextBoundary(), now, body)
	}

	return compileAttachmentsMIME(email, envelope, nextBoundary, now, body)
}

func compileAlternativeMIME(email Email, envelope emailEnvelope, boundary string, now time.Time, body io.Writer) error {
	if err := writeMessageHeaders(body, envelope, email.Subject, "multipart/alternative", boundary, now); err != nil {
		return err
	}

	writer := multipart.NewWriter(body)
	if err := writer.SetBoundary(boundary); err != nil {
		return fmt.Errorf("could not write email boundary: %w", err)
	}
	if err := writeBodyParts(writer, email.TextBody, email.HtmlBody); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("could not close multipart writer: %w", err)
	}

	return nil
}

func compileAttachmentsMIME(email EmailWithAttachments, envelope emailEnvelope, nextBoundary func() string, now time.Time, body io.Writer) error {
	mixedBoundary := nextBoundary()
	alternativeBoundary := nextBoundary()
	if err := writeMessageHeaders(body, envelope, email.Subject, "multipart/mixed", mixedBoundary, now); err != nil {
		return err
	}

	mixedWriter := multipart.NewWriter(body)
	if err := mixedWriter.SetBoundary(mixedBoundary); err != nil {
		return fmt.Errorf("could not write email boundary: %w", err)
	}
	if err := writeAlternativePart(mixedWriter, alternativeBoundary, email.TextBody, email.HtmlBody); err != nil {
		return err
	}
	for index := range email.Attachments {
		if err := writeAttachmentPart(mixedWriter, &email.Attachments[index]); err != nil {
			return err
		}
	}
	if err := mixedWriter.Close(); err != nil {
		return fmt.Errorf("could not close multipart writer: %w", err)
	}

	return nil
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
		return fmt.Errorf("could not close alternative email boundary: %w", err)
	}

	return nil
}

func writeAttachmentPart(mixedWriter *multipart.Writer, attachment *Attachment) error {
	mediaType, params, err := mime.ParseMediaType(attachment.ContentType)
	if err != nil {
		return fmt.Errorf("parse email attachment content type: %w", err)
	}
	params["name"] = attachment.Filename

	contentType, err := formatAttachmentHeaderValue("Content-Type", mediaType, params)
	if err != nil {
		return err
	}
	contentDisposition, err := formatAttachmentHeaderValue(
		"Content-Disposition",
		"attachment",
		map[string]string{"filename": attachment.Filename},
	)
	if err != nil {
		return err
	}

	attachmentHeader := textproto.MIMEHeader{}
	attachmentHeader.Set("Content-Type", contentType)
	attachmentHeader.Set("Content-Disposition", contentDisposition)
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
