package email

import (
	"fmt"
	"net/mail"
	"strings"

	"github.com/justtrackio/gosoline/pkg/funk"
)

type emailEnvelope struct {
	sender        *mail.Address
	recipients    []*mail.Address
	ccRecipients  []*mail.Address
	bccRecipients []*mail.Address
}

func (e emailEnvelope) recipientAddresses() []string {
	return addresses(e.recipients)
}

func (e emailEnvelope) ccAddresses() []string {
	return addresses(e.ccRecipients)
}

func (e emailEnvelope) bccAddresses() []string {
	return addresses(e.bccRecipients)
}

// deliveryAddresses lists every address the message is delivered to, including the bcc recipients missing from its headers.
func (e emailEnvelope) deliveryAddresses() []string {
	return append(append(e.recipientAddresses(), e.ccAddresses()...), e.bccAddresses()...)
}

func (e emailEnvelope) senderMailbox() string {
	return formatMailboxAddress(e.sender)
}

func (e emailEnvelope) recipientMailboxes() []string {
	return mailboxes(e.recipients)
}

func (e emailEnvelope) ccMailboxes() []string {
	return mailboxes(e.ccRecipients)
}

func (e emailEnvelope) bccMailboxes() []string {
	return mailboxes(e.bccRecipients)
}

func (e emailEnvelope) recipientHeader() string {
	return strings.Join(e.recipientMailboxes(), ", ")
}

func (e emailEnvelope) ccHeader() string {
	return strings.Join(e.ccMailboxes(), ", ")
}

// addresses and mailboxes return nil for an empty list, so optional destinations stay unset in provider requests.
func addresses(list []*mail.Address) []string {
	if len(list) == 0 {
		return nil
	}

	return funk.Map(list, func(mailbox *mail.Address) string {
		return mailbox.Address
	})
}

func mailboxes(list []*mail.Address) []string {
	if len(list) == 0 {
		return nil
	}

	return funk.Map(list, formatMailboxAddress)
}

func parseEmailEnvelope(fromAddress string, email Email) (emailEnvelope, error) {
	sender, err := mail.ParseAddress(fromAddress)
	if err != nil {
		return emailEnvelope{}, fmt.Errorf("format email sender: %w", err)
	}
	if len(email.Recipients) == 0 {
		return emailEnvelope{}, fmt.Errorf("format email recipients: recipient list is empty")
	}

	envelope := emailEnvelope{sender: sender}
	if envelope.recipients, err = parseAddressList(email.Recipients); err != nil {
		return emailEnvelope{}, fmt.Errorf("format email recipients: %w", err)
	}
	if envelope.ccRecipients, err = parseAddressList(email.CcRecipients); err != nil {
		return emailEnvelope{}, fmt.Errorf("format email cc recipients: %w", err)
	}
	if envelope.bccRecipients, err = parseAddressList(email.BccRecipients); err != nil {
		return emailEnvelope{}, fmt.Errorf("format email bcc recipients: %w", err)
	}

	return envelope, nil
}

func parseAddressList(list []string) ([]*mail.Address, error) {
	if len(list) == 0 {
		return nil, nil
	}

	return mail.ParseAddressList(strings.Join(list, ", "))
}
