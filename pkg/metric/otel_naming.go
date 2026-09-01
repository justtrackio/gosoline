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
		if (unicode.IsUpper(r) || unicode.IsDigit(r)) && i > 0 {
			prev := runes[i-1]

			var next rune
			if i+1 < len(runes) {
				next = runes[i+1]
			}

			startsNewWord := !unicode.IsUpper(prev) && !unicode.IsDigit(prev) || (next != 0 && unicode.IsLower(next))
			if startsNewWord && prev != '_' {
				b.WriteRune('_')
			}
		}

		b.WriteRune(unicode.ToLower(r))
	}

	return b.String()
}

// ToUcumUnitAndScale maps a CloudWatch StandardUnit to the UCUM unit OTEL reports it under, together
// with the factor a recorded value is multiplied by to arrive at that unit. Custom aggregation units
// are resolved to their base unit first, so a millisecond average reports seconds instead of a
// dimensionless 1. Durations resolve to seconds and byte counts to bytes, the base units both OTEL
// and Prometheus expect.
func ToUcumUnitAndScale(unit types.StandardUnit) (ucumUnit string, scale float64) {
	switch resolveBaseUnit(unit) {
	case types.StandardUnitCount:
		return "1", 1
	case types.StandardUnitSeconds:
		return "s", 1
	case types.StandardUnitMilliseconds:
		return "s", 1e-3
	case types.StandardUnitMicroseconds:
		return "s", 1e-6
	case types.StandardUnitBytes:
		return "By", 1
	case types.StandardUnitKilobytes:
		return "By", 1 << 10
	case types.StandardUnitMegabytes:
		return "By", 1 << 20
	case types.StandardUnitGigabytes:
		return "By", 1 << 30
	case types.StandardUnitTerabytes:
		return "By", 1 << 40
	case types.StandardUnitBits:
		return "bit", 1
	case types.StandardUnitPercent:
		return "%", 1
	case types.StandardUnitBytesSecond:
		return "By/s", 1
	case types.StandardUnitBitsSecond:
		return "bit/s", 1
	case types.StandardUnitCountSecond:
		return "1/s", 1
	default:
		return "1", 1
	}
}

// ToUcumUnit maps a CloudWatch StandardUnit to the closest UCUM unit used by OTEL. The unit is
// attached to the instrument; the Prometheus translation on the collector side appends the
// corresponding suffix (e.g. _seconds, _bytes) automatically.
func ToUcumUnit(unit types.StandardUnit) string {
	ucumUnit, _ := ToUcumUnitAndScale(unit)

	return ucumUnit
}

// unitScale returns the factor a value recorded in the given unit is multiplied by to arrive at the
// base unit both OTEL and Prometheus report it under.
func unitScale(unit types.StandardUnit) float64 {
	_, scale := ToUcumUnitAndScale(unit)

	return scale
}

// otelInstrumentUnit returns the unit an OTEL instrument for the given leaf and unit carries. A count
// of discrete things is reported as the UCUM annotation derived from the metric's own plural leaf, so
// a plural name and a non-unit unit can never disagree.
func otelInstrumentUnit(leaf string, unit types.StandardUnit) string {
	ucumUnit := ToUcumUnit(unit)

	if resolveBaseUnit(unit) != types.StandardUnitCount {
		return ucumUnit
	}

	if singular, ok := singularize(lastComponent(leaf)); ok {
		return "{" + singular + "}"
	}

	return ucumUnit
}

// singularize strips the plural suffix off a word, reporting whether the word was plural at all. It
// deliberately covers no more than a trailing `s` or `es`, and the conformance test asserts the
// result for every authored name so a surprise fails the build rather than a backend.
func singularize(word string) (string, bool) {
	if len(word) < 2 || !strings.HasSuffix(word, "s") || strings.HasSuffix(word, "ss") {
		return "", false
	}

	// `es` is only the plural suffix where the singular itself ends in a sibilant, so `batches`
	// singularizes to `batch` while `releases` singularizes to `release` rather than to `releas`.
	for _, suffix := range []string{"sses", "xes", "zes", "ches", "shes"} {
		if strings.HasSuffix(word, suffix) {
			return strings.TrimSuffix(word, "es"), true
		}
	}

	return strings.TrimSuffix(word, "s"), true
}

// lastComponent returns the last dot-delimited component of a canonical name.
func lastComponent(name string) string {
	if index := strings.LastIndex(name, "."); index >= 0 {
		return name[index+1:]
	}

	return name
}
