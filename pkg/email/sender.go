package email

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type SenderSetting struct {
	Type string `cfg:"type" default:"ses"`
}

type emailSettings struct {
	FromAddress string `cfg:"from_address"`
}

//go:generate go run github.com/vektra/mockery/v2 --name Sender
type Sender interface {
	SendEmail(ctx context.Context, email Email) error
}

func NewSender(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Sender, error) {
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
