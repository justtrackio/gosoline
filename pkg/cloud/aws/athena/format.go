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

	sb.WriteByte('\'')

	for i := 0; i < len(value); i++ {
		c := value[i]

		switch c {
		case '\\', '\'', '"':

			sb.WriteByte('\\')

			sb.WriteByte(c)
		case '\000':

			sb.WriteByte('\\')

			sb.WriteByte('0')
		case '\n':

			sb.WriteByte('\\')

			sb.WriteByte('n')
		case '\r':

			sb.WriteByte('\\')

			sb.WriteByte('r')
		case '\032':

			sb.WriteByte('\\')

			sb.WriteByte('Z')
		default:

			sb.WriteByte(c)
		}
	}

	sb.WriteByte('\'')

	return sb.String()
}
