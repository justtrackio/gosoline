package mapx

func SplitUnescapedDotN(s string, n int) []string {
	if n == 0 {
		return nil
	}

	var parts []string
	var current []rune
	escaped := false

	for _, r := range s {
		if r == '\\' && !escaped {
			escaped = true

			continue
		}

		if r == '.' && !escaped && (n < 0 || len(parts) < n-1) {
			parts = append(parts, string(current))
			current = current[:0]

			continue
		}

		if escaped {
			current = append(current, '\\')
			escaped = false
		}
		current = append(current, r)
	}

	parts = append(parts, string(current))

	return parts
}

// EscapeDots escapes all dots in the string with a backslash
// so they won't be treated as path separators
func EscapeDots(s string) string {
	var result []rune

	for _, r := range s {
		if r == '.' {
			result = append(result, '\\')
		}
		result = append(result, r)
	}

	return string(result)
}
