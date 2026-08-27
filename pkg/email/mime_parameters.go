package email

import (
	"fmt"
	"mime"
	"sort"
	"strconv"
	"strings"
)

const rfc2231ParameterChunkSize = 45

func formatAttachmentHeaderValue(header string, mediaType string, params map[string]string) (string, error) {
	formatted := mime.FormatMediaType(mediaType, params)
	folded, err := foldHeaderValue(header, formatted)
	if err == nil {
		return folded, nil
	}

	return foldHeaderValue(header, formatLongMediaTypeParameters(mediaType, params))
}

func formatLongMediaTypeParameters(mediaType string, params map[string]string) string {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var result strings.Builder
	result.WriteString(mediaType)
	for _, key := range keys {
		for index, value := range encodeRFC2231Parameter(params[key]) {
			result.WriteString("; ")
			result.WriteString(strings.ToLower(key))
			result.WriteString("*")
			result.WriteString(strconv.Itoa(index))
			result.WriteString("*=")
			result.WriteString(value)
		}
	}

	return result.String()
}

func encodeRFC2231Parameter(value string) []string {
	chunks := make([]string, 0, len(value)/rfc2231ParameterChunkSize+1)
	current := ""
	for _, byteValue := range []byte(value) {
		encoded := encodeRFC2231Byte(byteValue)
		if len(current)+len(encoded) > rfc2231ParameterChunkSize {
			chunks = append(chunks, current)
			current = ""
		}
		current += encoded
	}
	if current != "" || len(chunks) == 0 {
		chunks = append(chunks, current)
	}
	chunks[0] = "utf-8''" + chunks[0]

	return chunks
}

func encodeRFC2231Byte(value byte) string {
	if isRFC2231AttributeCharacter(value) {
		return string(value)
	}

	return fmt.Sprintf("%%%02X", value)
}

func isRFC2231AttributeCharacter(value byte) bool {
	return value >= 'a' && value <= 'z' ||
		value >= 'A' && value <= 'Z' ||
		value >= '0' && value <= '9' ||
		strings.ContainsRune("!#$&+-.^_`|~", rune(value))
}
