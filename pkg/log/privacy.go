package log

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// Redacted is the diagnostic representation of confidential information.
const Redacted = "[REDACTED]"

// Secret protects an explicitly confidential value from formatting and JSON
// serialization. Convert it to string only at the boundary that needs the value.
// It does not make arbitrary strings or enclosing objects confidential.
type Secret string

func (Secret) String() string   { return Redacted }
func (Secret) GoString() string { return Redacted }
func (Secret) Format(state fmt.State, _ rune) {
	if _, err := io.WriteString(state, Redacted); err != nil {
		return
	}
}
func (Secret) MarshalJSON() ([]byte, error) { return []byte(`"[REDACTED]"`), nil }

// IsSensitiveField identifies credential fields, not every configuration value.
// Additional application-specific secrets must be explicitly protected.
func IsSensitiveField(name string) bool {
	if dot := strings.LastIndexByte(name, '.'); dot >= 0 {
		name = name[dot+1:]
	}
	name = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}

		return -1
	}, name)
	for _, suffix := range []string{"password", "apikey", "clientsecret", "accesstoken", "refreshtoken", "sessiontoken"} {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	switch name {
	case "password", "passwd", "passphrase", "hashedpassword", "passwordhash",
		"apikey", "xapikey", "authorization", "xhashedauthorization", "cookie", "setcookie",
		"secret", "clientsecret", "signingsecret", "privatekey", "token", "jwt",
		"accesstoken", "refreshtoken", "idtoken", "sessiontoken", "googleidtoken",
		"verificationtoken", "resettoken", "code", "codeverifier", "state":
		return true
	default:
		return false
	}
}

// SanitizeURL removes all query, fragment and userinfo values from a diagnostic URL.
// Invalid URLs fail closed rather than returning the original string.
func SanitizeURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return Redacted
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.ForceQuery = false
	parsed.Fragment = ""
	parsed.RawFragment = ""

	return parsed.String()
}

// RedactError protects explicitly identified credentials in an error message,
// including URL and base64 forms. The original cause remains available to
// errors.Is/As for program control flow. Loggers must use SafeError at export.
// Do not pass ordinary configuration strings as confidential values.
func RedactError(err error, values ...string) error {
	if err == nil {
		return nil
	}

	return &redactedError{cause: err, values: secretForms(values)}
}

type redactedError struct {
	cause  error
	values []string
}

func (e *redactedError) Error() string { return redactText(e.cause.Error(), e.values) }
func (e *redactedError) Unwrap() error { return e.cause }
func (e *redactedError) Format(state fmt.State, _ rune) {
	if _, err := io.WriteString(state, e.Error()); err != nil {
		return
	}
}

// SafeError creates an export-only error when the chain contains RedactError.
// Its detached cause prevents Sentry or another sink traversing the original
// confidential chain. Keep the original error for errors.Is/As in application code.
func SafeError(err error) error {
	if err == nil {
		return nil
	}
	values := errorSecrets(err, 0)
	for _, value := range values {
		if value == "" {
			return errors.New("error chain omitted: privacy depth limit")
		}
	}
	if len(values) == 0 {
		return err
	}

	return errors.New(redactText(err.Error(), values))
}

func errorSecrets(err error, depth int) []string {
	if err == nil {
		return nil
	}
	if depth >= 32 {
		return []string{""}
	}
	var values []string
	if protected, ok := err.(*redactedError); ok {
		values = append(values, protected.values...)
	}
	switch wrapped := err.(type) {
	case interface{ Unwrap() []error }:
		for _, cause := range wrapped.Unwrap() {
			values = append(values, errorSecrets(cause, depth+1)...)
		}
	case interface{ Unwrap() error }:
		values = append(values, errorSecrets(wrapped.Unwrap(), depth+1)...)
	}

	return values
}

func secretForms(values []string) []string {
	unique := make(map[string]struct{})
	for _, value := range values {
		if value == "" {
			continue
		}
		quoted := strconv.Quote(value)
		for _, form := range []string{
			quoted[1 : len(quoted)-1], value, url.QueryEscape(value), url.PathEscape(value),
			base64.StdEncoding.EncodeToString([]byte(value)), base64.RawStdEncoding.EncodeToString([]byte(value)),
			base64.URLEncoding.EncodeToString([]byte(value)), base64.RawURLEncoding.EncodeToString([]byte(value)),
		} {
			unique[form] = struct{}{}
		}
	}
	forms := make([]string, 0, len(unique))
	for form := range unique {
		forms = append(forms, form)
	}
	sort.Slice(forms, func(i, j int) bool { return len(forms[i]) > len(forms[j]) })

	return forms
}

func redactText(text string, values []string) string {
	values = append([]string(nil), values...)
	sort.Slice(values, func(i, j int) bool { return len(values[i]) > len(values[j]) })
	for _, value := range values {
		if value != "" {
			text = replaceSecret(text, value)
		}
	}

	return text
}

func replaceSecret(text, value string) string {
	var out strings.Builder
	for offset := 0; offset < len(text); {
		relative := strings.Index(text[offset:], value)
		if relative < 0 {
			out.WriteString(text[offset:])

			break
		}
		start := offset + relative
		end := start + len(value)
		out.WriteString(text[offset:start])
		if (start > 0 && wordByte(text[start-1]) && wordByte(value[0])) ||
			(end < len(text) && wordByte(text[end]) && wordByte(value[len(value)-1])) {
			out.WriteString(value)
		} else {
			out.WriteString(Redacted)
		}
		offset = end
	}

	return out.String()
}

func wordByte(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '_' || b == '-'
}

// SanitizeFields copies common structured log containers and masks credential
// fields. It never mutates the caller's maps. Unknown objects are not inspected;
// use safe diagnostic projections rather than logging request/config objects.
func SanitizeFields(fields map[string]any) map[string]any {
	return sanitizeFields(fields, 0)
}

func sanitizeFields(fields map[string]any, depth int) map[string]any {
	out := make(map[string]any, len(fields))
	for key, value := range fields {
		if IsSensitiveField(key) || depth >= 16 {
			out[key] = Redacted
		} else {
			out[key] = sanitizeValue(value, depth+1)
		}
	}

	return out
}

func sanitizeValue(value any, depth int) any {
	switch typed := value.(type) {
	case Secret:
		return Redacted
	case error:
		return SafeError(typed)
	case map[string]any:
		return sanitizeFields(typed, depth)
	case Fields:
		return sanitizeFields(typed, depth)
	case map[string]string:
		fields := make(map[string]any, len(typed))
		for key, item := range typed {
			fields[key] = item
		}

		return sanitizeFields(fields, depth)
	case []any:
		if depth >= 16 {
			return Redacted
		}
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = sanitizeValue(item, depth+1)
		}

		return out
	default:
		return value
	}
}
