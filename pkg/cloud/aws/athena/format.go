package athena

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/justtrackio/gosoline/pkg/coffin"
)

var replaceRegExp = regexp.MustCompile(`\$\d+`)

// ReplaceDollarPlaceholders replaces $1, $2, etc. with the corresponding element from params. You have to supply
// an additional escape function which converts each parameter to a string safe to embed in the query.
func ReplaceDollarPlaceholders(query string, args []any) (sql string, err error) {
	defer func() {
		if err != nil {
			return
		}

		err = coffin.ResolveRecovery(recover())
	}()

	return replaceRegExp.ReplaceAllStringFunc(query, func(s string) string {
		index, err := strconv.ParseInt(s[1:], 10, 64)
		if err != nil || index < 1 || index > int64(len(args)) {
			return s
		}

		arg := args[index-1]

		switch a := arg.(type) {
		case bool:
			return strconv.FormatBool(a)
		case string:
			return EscapeString(a)
		case fmt.Stringer:
			return EscapeString(a.String())
		case []byte:
			return EscapeString(string(a))
		case int, int8, int16, int32, int64:
			return fmt.Sprintf("%d", a)
		case uint, uint8, uint16, uint32, uint64:
			return fmt.Sprintf("%d", a)
		case float32, float64:
			return fmt.Sprintf("%g", a)
		}

		panic(fmt.Sprintf("unsupported type %T for arg[%d]: %v", arg, index-1, arg))
	}), nil
}

// EscapeString converts a string value to something you can embed in an SQL query (if you can't use parametrized queries).
// The string is surrounded by single quotes, so you don't need to take care of that yourself.
func EscapeString(value string) string {
	var sb strings.Builder

	//nolint:errcheck // WriteByte always returns nil
	sb.WriteByte('\'')

	for i := 0; i < len(value); i++ {
		c := value[i]

		switch c {
		case '\\', '\'', '"':
			//nolint:errcheck // WriteByte always returns nil
			sb.WriteByte('\\')
			//nolint:errcheck // WriteByte always returns nil
			sb.WriteByte(c)
		case '\000':
			//nolint:errcheck // WriteByte always returns nil
			sb.WriteByte('\\')
			//nolint:errcheck // WriteByte always returns nil
			sb.WriteByte('0')
		case '\n':
			//nolint:errcheck // WriteByte always returns nil
			sb.WriteByte('\\')
			//nolint:errcheck // WriteByte always returns nil
			sb.WriteByte('n')
		case '\r':
			//nolint:errcheck // WriteByte always returns nil
			sb.WriteByte('\\')
			//nolint:errcheck // WriteByte always returns nil
			sb.WriteByte('r')
		case '\032':
			//nolint:errcheck // WriteByte always returns nil
			sb.WriteByte('\\')
			//nolint:errcheck // WriteByte always returns nil
			sb.WriteByte('Z')
		default:
			//nolint:errcheck // WriteByte always returns nil
			sb.WriteByte(c)
		}
	}

	//nolint:errcheck // WriteByte always returns nil
	sb.WriteByte('\'')

	return sb.String()
}
