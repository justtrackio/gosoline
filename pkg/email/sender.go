package email

import (
	"context"
	"fmt"
	"io"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type SenderSetting struct {
	Type string `cfg:"type" default:"ses"`
}

const defaultMaxEncodedMessageSize = 10 * 1024 * 1024

type emailSettings struct {
	FromAddress           string `cfg:"from_address"`
	MaxEncodedMessageSize int    `cfg:"max_encoded_message_size" default:"10485760"`
}

func (s emailSettings) validate() error {
	if s.MaxEncodedMessageSize <= 0 {
		return fmt.Errorf("max encoded message size must be positive, got %d", s.MaxEncodedMessageSize)
	}

	return nil
}

// encodedMessageSizeWriter fails the write which would push the encoded message past limit, so that a message is rejected
// while it is being encoded instead of after it has been fully buffered.
type encodedMessageSizeWriter struct {
	writer  io.Writer
	limit   int
	written int
}

func (w *encodedMessageSizeWriter) Write(data []byte) (int, error) {
	if len(data) > w.limit-w.written {
		return 0, fmt.Errorf("encoded email message size exceeds configured limit of %d bytes", w.limit)
	}

	written, err := w.writer.Write(data)
	w.written += written
	if err == nil && written != len(data) {
		return written, io.ErrShortWrite
	}

	return written, err
}

//go:generate go run github.com/vektra/mockery/v2 --name Sender
type Sender interface {
	SendEmail(ctx context.Context, email Email) error
}

// SenderWithAttachments supports messages with one or more attachments.
//
//go:generate go run github.com/vektra/mockery/v2 --name SenderWithAttachments
type SenderWithAttachments interface {
	Sender

	SendEmailWithAttachments(ctx context.Context, email EmailWithAttachments) error
}

func NewSender(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Sender, error) {
	return NewSenderWithAttachments(ctx, config, logger, name)
}

func NewSenderWithAttachments(ctx context.Context, config cfg.Config, logger log.Logger, name string) (SenderWithAttachments, error) {
	settings := &SenderSetting{}
	if err := config.UnmarshalKey(fmt.Sprintf("email.%s", name), settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sender settings: %w", err)
	}

	switch settings.Type {
	case "ses":
		return NewSesSender(ctx, config, logger, name)
	case "smtp":
		return NewSmtpSender(config, name)
	default:
		return nil, fmt.Errorf("unknown email sender type: %q", settings.Type)
	}
}
