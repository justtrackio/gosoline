package metric

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/stretchr/testify/assert"
)

func TestFormatOtelMetricName(t *testing.T) {
	cases := map[string]string{
		"ApiRequestCount":        "api_request_count",
		"stream.ConsumerError":   "stream.consumer_error",
		"db-query/duration":      "db.query.duration",
		"HTTPServer":             "http_server",
		"apiV2":                  "api_v2",
		"already_snake":          "already_snake",
		"stream.consumer.errors": "stream.consumer.errors",
		"Mixed Sep-Here":         "mixed.sep.here",
		"HTTPStatus2XXPerRoute":  "http_status_2xx_per_route",
	}

	for input, expected := range cases {
		assert.Equal(t, expected, FormatOtelMetricName(input), "input %q", input)
	}
}

func TestToUcumUnitAndScale(t *testing.T) {
	cases := map[types.StandardUnit]struct {
		unit  string
		scale float64
	}{
		types.StandardUnitCount:        {unit: "1", scale: 1},
		types.StandardUnitSeconds:      {unit: "s", scale: 1},
		types.StandardUnitMilliseconds: {unit: "s", scale: 1e-3},
		types.StandardUnitMicroseconds: {unit: "s", scale: 1e-6},
		types.StandardUnitBytes:        {unit: "By", scale: 1},
		types.StandardUnitKilobytes:    {unit: "By", scale: 1024},
		types.StandardUnitPercent:      {unit: "%", scale: 1},
		types.StandardUnitBitsSecond:   {unit: "bit/s", scale: 1},
		types.StandardUnitNone:         {unit: "1", scale: 1},
		// a custom aggregation unit resolves to its base unit rather than falling through to 1
		UnitMillisecondsAverage: {unit: "s", scale: 1e-3},
		UnitMillisecondsMaximum: {unit: "s", scale: 1e-3},
		UnitCountAverage:        {unit: "1", scale: 1},
		UnitSecondsMinimum:      {unit: "s", scale: 1},
	}

	for unit, expected := range cases {
		ucumUnit, scale := ToUcumUnitAndScale(unit)

		assert.Equal(t, expected.unit, ucumUnit, "unit %q", unit)
		assert.Equal(t, expected.scale, scale, "unit %q", unit)
		assert.Equal(t, expected.unit, ToUcumUnit(unit), "unit %q", unit)
	}
}

func TestOtelInstrumentUnit(t *testing.T) {
	tests := map[string]struct {
		leaf     string
		unit     types.StandardUnit
		expected string
	}{
		"plural leaf yields a non unit": {
			leaf:     "errors",
			unit:     UnitCount,
			expected: "{error}",
		},
		"dotted plural leaf uses its last component": {
			leaf:     "client.consumed.messages",
			unit:     UnitCount,
			expected: "{message}",
		},
		"plural leaf ending in a sibilant strips es": {
			leaf:     "batches",
			unit:     UnitCount,
			expected: "{batch}",
		},
		"plural leaf whose singular ends in e keeps it": {
			leaf:     "rate_limit.releases",
			unit:     UnitCount,
			expected: "{release}",
		},
		"singular leaf keeps the dimensionless unit": {
			leaf:     "item.count",
			unit:     UnitCount,
			expected: "1",
		},
		"a leaf that only looks plural keeps the dimensionless unit": {
			leaf:     "loss",
			unit:     UnitCount,
			expected: "1",
		},
		"a measured unit is never replaced by a non unit": {
			leaf:     "request.duration",
			unit:     UnitMilliseconds,
			expected: "s",
		},
		"a plural leaf measuring bytes keeps its unit": {
			leaf:     "produce.batch.size",
			unit:     UnitBytes,
			expected: "By",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, otelInstrumentUnit(tt.leaf, tt.unit))
		})
	}
}

// TestInferKindIsConsistentAcrossWriters pins down that a datum without a declared instrument type is
// classified by its unit alone, so the prometheus and otel writers can not disagree on it.
func TestInferKindIsConsistentAcrossWriters(t *testing.T) {
	tests := map[types.StandardUnit]kind{
		UnitCount:                      kindCounter,
		UnitCountAverage:               kindCounter,
		UnitMilliseconds:               kindHistogram,
		UnitMillisecondsAverage:        kindHistogram,
		UnitMillisecondsMaximum:        kindHistogram,
		UnitSeconds:                    kindHistogram,
		UnitSecondsAverage:             kindHistogram,
		types.StandardUnitMicroseconds: kindHistogram,
		UnitBytes:                      kindGauge,
		types.StandardUnitNone:         kindGauge,
	}

	for unit, expected := range tests {
		datum := &Datum{MetricName: "leaf", Unit: unit}

		assert.Equal(t, expected, inferKind(unit), "unit %q", unit)
		assert.Equal(t, expected, effectiveKind(datum), "unit %q", unit)
	}
}

// TestEffectiveKindPrefersTheDeclaredType pins down that a declared instrument type is never
// overridden by unit based inference.
func TestEffectiveKindPrefersTheDeclaredType(t *testing.T) {
	datum := &Datum{
		MetricName: "leaf",
		Unit:       UnitCount,
		Kind:       KindGauge.Build(),
	}

	assert.Equal(t, kindGauge, effectiveKind(datum))
}
