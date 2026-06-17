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
	}

	for input, expected := range cases {
		assert.Equal(t, expected, FormatOtelMetricName(input), "input %q", input)
	}
}

func TestToUcumUnit(t *testing.T) {
	cases := map[types.StandardUnit]string{
		types.StandardUnitCount:        "1",
		types.StandardUnitSeconds:      "s",
		types.StandardUnitMilliseconds: "ms",
		types.StandardUnitBytes:        "By",
		types.StandardUnitPercent:      "%",
		types.StandardUnitBitsSecond:   "bit/s",
		types.StandardUnitNone:         "1",
	}

	for unit, expected := range cases {
		assert.Equal(t, expected, ToUcumUnit(unit), "unit %q", unit)
	}
}
