package email

import (
	"errors"
	"mime"
	"strings"
)

// Attachment is a file attached to an email message.
type Attachment struct {
	Filename    string
	ContentType string
	Content     []byte
}

// Email is an email message without attachments.
type Email struct {
	Recipients []string
	Subject    string
	TextBody   *string
	HtmlBody   *string
}

// EmailWithAttachments is an email message with one or more attachments.
type EmailWithAttachments struct {
	Email
	Attachments []Attachment
}

func (e Email) validate() error {
	if e.HtmlBody == nil && e.TextBody == nil {
		return errors.New("email body cannot be empty")
	}
	if strings.ContainsAny(e.Subject, "\r\n") {
		return errors.New("email subject cannot contain line breaks")
	}

	return nil
}

func (e EmailWithAttachments) validate() error {
	if err := e.Email.validate(); err != nil {
		return err
	}
	if len(e.Attachments) == 0 {
		return errors.New("email attachments cannot be empty")
	}
	for _, attachment := range e.Attachments {
		if err := attachment.validate(); err != nil {
			return err
		}
	}

	return nil
}

func (a Attachment) validate() error {
	if strings.TrimSpace(a.Filename) == "" {
		return errors.New("email attachment filename cannot be empty")
	}
	if strings.ContainsAny(a.Filename, "\r\n") {
		return errors.New("email attachment filename cannot contain line breaks")
	}
	if err := validateMediaType(a.ContentType); err != nil {
		return err
	}

	return nil
}

func validateMediaType(contentType string) error {
	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		return errors.New("email attachment content type cannot be empty")
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return errors.New("email attachment content type is invalid")
	}
	mainType, subType, ok := strings.Cut(mediaType, "/")
	if !ok || mainType == "" || subType == "" {
		return errors.New("email attachment content type is invalid")
	}

	return nil
}
