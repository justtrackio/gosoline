package metric

import (
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

// otelGosolinePrefix namespaces every metric gosoline invents, so gosoline never defines a name
// inside a namespace an OpenTelemetry semantic convention owns.
const otelGosolinePrefix = "gosoline."

// semanticConventionNamespaces holds the namespaces an OpenTelemetry semantic convention owns.
// Membership is matched exactly, so `db.repo` is gosoline's while `db.client` is the convention's.
var semanticConventionNamespaces = map[string]struct{}{
	"db.client":   {},
	"http.client": {},
	"http.server": {},
	"messaging":   {},
	"rpc.server":  {},
}

// gosolineNamesInSemanticConventionNamespaces holds the canonical names gosoline invents inside a
// namespace a semantic convention owns. They carry the gosoline prefix despite their namespace.
var gosolineNamesInSemanticConventionNamespaces = map[string]struct{}{
	"db.client.connections":         {},
	"http.server.connection.count":  {},
	"http.server.rejected.requests": {},
}

// canonicalName joins a canonical namespace and leaf into the dotted canonical form. A leaf without
// a namespace is its own canonical name.
func canonicalName(namespace, leaf string) string {
	if namespace == "" {
		return leaf
	}

	return namespace + "." + leaf
}

// CloudWatchMetricName renders a canonical namespace and leaf into the name the CloudWatch writer
// exports the metric under. Reading a gosoline metric back out of CloudWatch, as the per-runner
// metric calculator does, needs the exported name rather than the authored one.
func CloudWatchMetricName(namespace, leaf string) string {
	return renderCloudWatchName(namespace, leaf)
}

// renderCloudWatchName renders a canonical namespace and leaf into a single PascalCase name by
// splitting on both canonical separators, capitalising every word and concatenating them without a
// separator. It adds no gosoline prefix, because the CloudWatch namespace already identifies the
// application. A datum without a namespace keeps its name verbatim, so metrics authored outside
// gosoline are exported unchanged.
func renderCloudWatchName(namespace, leaf string) string {
	if namespace == "" {
		return leaf
	}

	words := strings.FieldsFunc(canonicalName(namespace, leaf), isCanonicalSeparator)
	rendered := make([]string, 0, len(words))

	for _, word := range words {
		rendered = append(rendered, capitalizeFirst(word))
	}

	return strings.Join(rendered, "")
}

// renderPrometheusName renders a canonical namespace into a Prometheus subsystem and a canonical
// leaf into a Prometheus metric name, both by replacing dots with underscores. The name carries the
// base-unit suffix and the `_total` suffix the Prometheus convention requires. No gosoline prefix is
// added, because the application-derived Prometheus namespace already supplies the single
// application prefix the convention calls for. A datum without a namespace keeps its name verbatim.
func renderPrometheusName(namespace, leaf string, unit types.StandardUnit, metricKind kind) (subsystem string, name string) {
	if namespace == "" {
		return "", leaf
	}

	name = dotsToUnderscores(leaf) + prometheusUnitSuffix(unit) + prometheusCounterSuffix(metricKind)

	return dotsToUnderscores(namespace), name
}

// renderOtelName renders a canonical namespace and leaf into the dotted canonical form, carrying no
// application prefix because identity is a resource attribute. The name is prefixed with `gosoline.`
// unless the metric corresponds to an OpenTelemetry semantic convention. A datum without a namespace
// keeps its normalized name, so metrics authored outside gosoline are exported unchanged.
func renderOtelName(namespace, leaf string) string {
	if namespace == "" {
		return FormatOtelMetricName(leaf)
	}

	name := canonicalName(namespace, leaf)
	if isSemanticConvention(namespace, name) {
		return name
	}

	return otelGosolinePrefix + name
}

// isSemanticConvention reports whether a canonical name is defined by an OpenTelemetry semantic
// convention rather than invented by gosoline.
func isSemanticConvention(namespace, name string) bool {
	if _, ok := gosolineNamesInSemanticConventionNamespaces[name]; ok {
		return false
	}

	_, ok := semanticConventionNamespaces[namespace]

	return ok
}

// prometheusUnitSuffix returns the plural base-unit suffix the Prometheus convention requires for a
// unit. A count of discrete things carries no unit suffix - it is suffixed with `_total` instead.
func prometheusUnitSuffix(unit types.StandardUnit) string {
	switch resolveBaseUnit(unit) {
	case types.StandardUnitSeconds, types.StandardUnitMilliseconds, types.StandardUnitMicroseconds:
		return "_seconds"
	case types.StandardUnitBytes,
		types.StandardUnitKilobytes,
		types.StandardUnitMegabytes,
		types.StandardUnitGigabytes,
		types.StandardUnitTerabytes:
		return "_bytes"
	default:
		return ""
	}
}

// prometheusCounterSuffix returns the `_total` suffix the Prometheus convention requires for a
// monotonically accumulating counter.
func prometheusCounterSuffix(metricKind kind) string {
	if metricKind == kindCounter {
		return "_total"
	}

	return ""
}

// renderPrometheusLabelName renders a canonical dimension key into a valid Prometheus label name by
// replacing dots with underscores, the same way the canonical metric name is rendered.
func renderPrometheusLabelName(key string) string {
	return dotsToUnderscores(key)
}

func isCanonicalSeparator(r rune) bool {
	return r == '.' || r == '_'
}

func capitalizeFirst(word string) string {
	if word == "" {
		return word
	}

	return strings.ToUpper(word[:1]) + word[1:]
}

func dotsToUnderscores(name string) string {
	return strings.ReplaceAll(name, ".", "_")
}
