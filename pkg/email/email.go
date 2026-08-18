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

type Email struct {
	Recipients []string
	Subject    string
	TextBody   *string
	HtmlBody   *string
	Attachment *Attachment
}

func (e Email) validate() error {
	if e.HtmlBody == nil && e.TextBody == nil {
		return errors.New("email body cannot be empty")
	}
	if strings.ContainsAny(e.Subject, "\r\n") {
		return errors.New("email subject cannot contain line breaks")
	}
	if e.Attachment == nil {
		return nil
	}
	if strings.TrimSpace(e.Attachment.Filename) == "" {
		return errors.New("email attachment filename cannot be empty")
	}
	if strings.ContainsAny(e.Attachment.Filename, "\r\n") {
		return errors.New("email attachment filename cannot contain line breaks")
	}
	if err := validateMediaType(e.Attachment.ContentType); err != nil {
		return err
	}
	if len(e.Attachment.Content) == 0 {
		return errors.New("email attachment content cannot be empty")
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
