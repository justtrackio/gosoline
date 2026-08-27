package email

import (
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	mimeLineBreak = "\r\n"
	// RFC 5322 section 2.1.1 recommends limiting header lines to 78 characters.
	recommendedHeaderLineLength = 78
	// RFC 5322 section 2.1.1 requires header lines to be no longer than 998 characters.
	maximumHeaderLineLength = 998
	// 45 raw bytes encode to a 72-character RFC 2047 base64 encoded word, leaving room for folding.
	encodedWordMaximumPayload = 45
)

type messageHeader struct {
	name  string
	value string
}

// formatMailboxAddress renders a mailbox as a header value. A mailbox without a display name stays a plain address, which is
// also what a provider rendering the message on our behalf expects. mail.Address.String already encodes display names which
// need it, but it emits a single encoded word which can exceed the line length limits. Only in that case do we re-encode the
// display name into a sequence of encoded words, which foldHeader can spread over multiple lines. A long display name which
// does not need encoding is left readable, foldHeader wraps it at its whitespace.
func formatMailboxAddress(mailbox *mail.Address) string {
	if mailbox.Name == "" {
		return mailbox.Address
	}

	formatted := mailbox.String()
	if len(formatted) <= recommendedHeaderLineLength || !needsEncodedWords(mailbox.Name) {
		return formatted
	}

	return encodeHeaderWords(mailbox.Name) + " <" + mailbox.Address + ">"
}

// writeMessageHeaders writes the top level headers of a message. It takes the envelope instead of preformatted values so
// that the From and To headers cannot drift apart from the addresses the message is actually sent to.
func writeMessageHeaders(body io.Writer, envelope emailEnvelope, subject string, contentType string, boundary string, now time.Time) error {
	headers := []messageHeader{
		{name: "Date", value: now.Format(time.RFC1123Z)},
		{name: "From", value: envelope.senderMailbox()},
		{name: "To", value: envelope.recipientHeader()},
		{name: "Subject", value: encodeHeader(subject)},
		{name: "MIME-Version", value: "1.0"},
		{name: "Content-Type", value: mime.FormatMediaType(contentType, map[string]string{"boundary": boundary})},
	}

	return writeHeaders(body, headers)
}

func writeHeaders(body io.Writer, headers []messageHeader) error {
	for _, header := range headers {
		folded, err := foldHeader(header.name, header.value)
		if err != nil {
			return err
		}
		if _, err := io.WriteString(body, folded+mimeLineBreak); err != nil {
			return fmt.Errorf("could not write email header: %w", err)
		}
	}
	if _, err := io.WriteString(body, mimeLineBreak); err != nil {
		return fmt.Errorf("could not write email header: %w", err)
	}

	return nil
}

func foldHeaderValue(name string, value string) (string, error) {
	folded, err := foldHeader(name, value)
	if err != nil {
		return "", err
	}

	return strings.TrimPrefix(folded, name+": "), nil
}

func foldHeader(name string, value string) (string, error) {
	if strings.ContainsAny(value, "\r\n") {
		return "", fmt.Errorf("email %s header cannot contain line breaks", strings.ToLower(name))
	}
	if value == "" {
		return name + ": ", nil
	}

	var result strings.Builder
	result.WriteString(name)
	result.WriteString(":")
	lineLength := len(name) + 1
	pendingWhitespace := " "

	for value != "" {
		whitespaceLength := strings.IndexFunc(value, func(r rune) bool {
			return r != ' ' && r != '\t'
		})
		if whitespaceLength == -1 {
			pendingWhitespace += value

			break
		}
		if whitespaceLength > 0 {
			pendingWhitespace += value[:whitespaceLength]
			value = value[whitespaceLength:]
		}

		wordLength := strings.IndexFunc(value, func(r rune) bool {
			return r == ' ' || r == '\t'
		})
		if wordLength == -1 {
			wordLength = len(value)
		}
		word := value[:wordLength]
		value = value[wordLength:]

		var err error
		lineLength, err = appendFoldedHeaderWord(&result, name, lineLength, pendingWhitespace, word)
		if err != nil {
			return "", err
		}
		pendingWhitespace = ""
	}

	if pendingWhitespace != "" {
		if lineLength+len(pendingWhitespace) > maximumHeaderLineLength {
			return "", fmt.Errorf("email %s header line exceeds %d bytes", strings.ToLower(name), maximumHeaderLineLength)
		}
		result.WriteString(pendingWhitespace)
	}

	return result.String(), nil
}

func appendFoldedHeaderWord(result *strings.Builder, name string, lineLength int, pendingWhitespace string, word string) (int, error) {
	if lineLength+len(pendingWhitespace)+len(word) > recommendedHeaderLineLength && lineLength > len(name)+1 {
		if len(word)+1 > maximumHeaderLineLength {
			return 0, fmt.Errorf("email %s header line exceeds %d bytes", strings.ToLower(name), maximumHeaderLineLength)
		}
		result.WriteString(mimeLineBreak)
		result.WriteByte(' ')
		lineLength = 1
		if len(pendingWhitespace) > 1 {
			result.WriteString(pendingWhitespace[1:])
			lineLength += len(pendingWhitespace) - 1
		}
	} else {
		result.WriteString(pendingWhitespace)
		lineLength += len(pendingWhitespace)
	}

	if lineLength+len(word) > maximumHeaderLineLength {
		return 0, fmt.Errorf("email %s header line exceeds %d bytes", strings.ToLower(name), maximumHeaderLineLength)
	}
	result.WriteString(word)

	return lineLength + len(word), nil
}

func encodeHeader(data string) string {
	if needsEncodedWords(data) {
		return encodeHeaderWords(data)
	}

	return data
}

// needsEncodedWords reports whether value has to be encoded as RFC 2047 encoded words, either because it is not plain ASCII
// or because it contains a token which no amount of folding can fit into a header line.
func needsEncodedWords(value string) bool {
	return containsNonASCII(value) || hasLongHeaderToken(value)
}

func containsNonASCII(value string) bool {
	return !utf8.ValidString(value) || strings.IndexFunc(value, func(r rune) bool {
		return r > 127
	}) != -1
}

func hasLongHeaderToken(value string) bool {
	for _, token := range strings.Fields(value) {
		if len(token) > maximumHeaderLineLength-1 {
			return true
		}
	}

	return false
}

func encodeHeaderWords(value string) string {
	words := make([]string, 0, len(value)/encodedWordMaximumPayload+1)
	for value != "" {
		length := encodedWordLength(value)
		words = append(words, "=?UTF-8?B?"+base64.StdEncoding.EncodeToString([]byte(value[:length]))+"?=")
		value = value[length:]
	}

	return strings.Join(words, " ")
}

func encodedWordLength(value string) int {
	length := 0
	for length < len(value) {
		_, size := utf8.DecodeRuneInString(value[length:])
		if size == 0 || length+size > encodedWordMaximumPayload {
			break
		}
		length += size
	}
	if length == 0 {
		return min(len(value), encodedWordMaximumPayload)
	}

	return length
}
