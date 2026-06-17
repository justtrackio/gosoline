package metric

import (
	"strings"
	"unicode"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

// FormatOtelMetricName normalizes a gosoline metric name to the OTEL naming standard:
// lowercase, dot-delimited namespaces, snake_case within each segment. Identity (env, app,
// team) must NOT be part of the name — it belongs in resource attributes. Units must NOT be
// part of the name either — they are set on the instrument (see ToUcumUnit).
//
// Examples:
//
//	"ApiRequestCount"        -> "api_request_count"
//	"stream.ConsumerError"   -> "stream.consumer_error"
//	"db-query/duration"      -> "db.query.duration"
func FormatOtelMetricName(name string) string {
	separators := strings.NewReplacer(" ", ".", "-", ".", "/", ".", ":", ".", "\\", ".")
	normalized := separators.Replace(name)

	parts := strings.Split(normalized, ".")
	out := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		out = append(out, strings.ToLower(camelToSnake(part)))
	}

	return strings.Join(out, ".")
}

// camelToSnake converts a single name segment from camelCase/PascalCase to snake_case,
// keeping acronym runs together (e.g. "HTTPServer" -> "http_server", "apiV2" -> "api_v2").
// NOTE: We intentionally avoid github.com/iancoleman/strcase.ToSnake here because it splits
// on digit boundaries (e.g. "apiV2" -> "api_v_2"), which is undesirable for metric names.
func camelToSnake(segment string) string {
	runes := []rune(segment)
	var b strings.Builder

	for i, r := range runes {
		if unicode.IsUpper(r) && i > 0 {
			prev := runes[i-1]

			var next rune
			if i+1 < len(runes) {
				next = runes[i+1]
			}

			startsNewWord := !unicode.IsUpper(prev) || (next != 0 && unicode.IsLower(next))
			if startsNewWord && prev != '_' {
				b.WriteRune('_')
			}
		}

		b.WriteRune(unicode.ToLower(r))
	}

	return b.String()
}

// ToUcumUnit maps a CloudWatch StandardUnit to the closest UCUM unit used by OTEL. The unit is
// attached to the instrument; the Prometheus translation on the collector side appends the
// corresponding suffix (e.g. _seconds, _bytes) automatically.
func ToUcumUnit(unit types.StandardUnit) string {
	switch unit {
	case types.StandardUnitCount:
		return "1"
	case types.StandardUnitSeconds:
		return "s"
	case types.StandardUnitMilliseconds:
		return "ms"
	case types.StandardUnitMicroseconds:
		return "us"
	case types.StandardUnitBytes:
		return "By"
	case types.StandardUnitKilobytes:
		return "kBy"
	case types.StandardUnitMegabytes:
		return "MBy"
	case types.StandardUnitGigabytes:
		return "GBy"
	case types.StandardUnitTerabytes:
		return "TBy"
	case types.StandardUnitBits:
		return "bit"
	case types.StandardUnitPercent:
		return "%"
	case types.StandardUnitBytesSecond:
		return "By/s"
	case types.StandardUnitBitsSecond:
		return "bit/s"
	case types.StandardUnitCountSecond:
		return "1/s"
	default:
		return "1"
	}
}
