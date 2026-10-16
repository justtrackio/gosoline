package metric

import (
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

// otelGosolinePrefix namespaces every metric gosoline emits. No gosoline metric claims a canonical
// OpenTelemetry semantic-convention name: the conventions are followed for grammar, units and
// attribute shape, while the names stay gosoline's own, so nothing gosoline exports can be mistaken
// for the convention's metric.
const otelGosolinePrefix = "gosoline."

// canonicalName joins a canonical namespace and leaf into the dotted canonical form. A leaf without
// a namespace is its own canonical name.
func canonicalName(namespace, leaf string) string {
	if namespace == "" {
		return leaf
	}

	return namespace + "." + leaf
}

// renderCloudWatchName renders a canonical namespace and leaf into a single PascalCase name by
// splitting on both canonical separators, capitalising every word and concatenating them without a
// separator. It adds no gosoline prefix, because the CloudWatch namespace already identifies the
// application. A datum without a namespace is rendered from its leaf alone, so a leaf carrying
// canonical separators is converted exactly as it would be underneath a namespace.
func renderCloudWatchName(namespace, leaf string) string {
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
// added here, because the writer's own namespace already supplies the single prefix the convention
// calls for. A datum without a namespace yields an empty subsystem, and its leaf is converted exactly
// as it would be underneath a namespace, because a dot is not a valid character in a Prometheus
// metric name regardless of where the name came from.
func renderPrometheusName(namespace, leaf string, unit types.StandardUnit, metricKind kind) (subsystem string, name string) {
	return dotsToUnderscores(namespace), dotsToUnderscores(leaf) + prometheusUnitSuffix(unit, metricKind)
}

// renderOtelName renders a canonical namespace and leaf into the dotted canonical form, carrying no
// application prefix because identity is a resource attribute. Every name is prefixed with
// `gosoline.`, so no exported name collides with an OpenTelemetry semantic convention. A datum without
// a namespace keeps its normalized name, so metrics authored outside gosoline are exported unchanged.
func renderOtelName(namespace, leaf string) string {
	if namespace == "" {
		return FormatOtelMetricName(leaf)
	}

	return otelGosolinePrefix + canonicalName(namespace, leaf)
}

// prometheusUnitSuffix returns the suffix the Prometheus convention requires for a datum: the plural
// base-unit suffix of its unit, followed by `_total` when the datum is a monotonically accumulating
// counter. A count of discrete things carries no unit suffix - `_total` alone conveys it.
func prometheusUnitSuffix(unit types.StandardUnit, metricKind kind) string {
	var suffix string

	switch resolveBaseUnit(unit) {
	case types.StandardUnitSeconds, types.StandardUnitMilliseconds, types.StandardUnitMicroseconds:
		suffix = "_seconds"
	case types.StandardUnitBytes,
		types.StandardUnitKilobytes,
		types.StandardUnitMegabytes,
		types.StandardUnitGigabytes,
		types.StandardUnitTerabytes:
		suffix = "_bytes"
	}

	if metricKind == kindCounter {
		suffix += "_total"
	}

	return suffix
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
