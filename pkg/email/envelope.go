package email

import (
	"fmt"
	"net/mail"
	"strings"

	"github.com/justtrackio/gosoline/pkg/funk"
)

type emailEnvelope struct {
	sender     *mail.Address
	recipients []*mail.Address
}

func (e emailEnvelope) recipientAddresses() []string {
	return funk.Map(e.recipients, func(recipient *mail.Address) string {
		return recipient.Address
	})
}

func (e emailEnvelope) senderMailbox() string {
	return formatMailboxAddress(e.sender)
}

func (e emailEnvelope) recipientMailboxes() []string {
	return funk.Map(e.recipients, formatMailboxAddress)
}

func (e emailEnvelope) recipientHeader() string {
	return strings.Join(e.recipientMailboxes(), ", ")
}

func parseEmailEnvelope(fromAddress string, recipients []string) (emailEnvelope, error) {
	sender, err := mail.ParseAddress(fromAddress)
	if err != nil {
		return emailEnvelope{}, fmt.Errorf("format email sender: %w", err)
	}
	if len(recipients) == 0 {
		return emailEnvelope{}, fmt.Errorf("format email recipients: recipient list is empty")
	}

	parsedRecipients, err := mail.ParseAddressList(strings.Join(recipients, ", "))
	if err != nil {
		return emailEnvelope{}, fmt.Errorf("format email recipients: %w", err)
	}

	return emailEnvelope{sender: sender, recipients: parsedRecipients}, nil
}
