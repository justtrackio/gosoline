package metric

import (
	"time"
)

const (
	kindDefault   kind = ""
	kindTotal     kind = "total"
	kindCounter   kind = "counter"
	kindGauge     kind = "gauge"
	kindHistogram kind = "histogram"
	kindSummary   kind = "summary"
)

var (
	// KindDefault is the zero value for a Kind. The prometheus writer will decide based on your unit what kind of metric you are writing.
	KindDefault = Kind{
		kind: kindDefault,
	}
	// KindTotal marks a metric as a summary metric which will be omitted on prometheus. Other writers (like CloudWatch) will still write
	// the metric, but for prometheus writing such a metric is not necessary.
	KindTotal = Kind{
		kind: kindTotal,
	}
	// KindCounter is the starting builder instance for counter metrics in prometheus (see also https://prometheus.io/docs/concepts/metric_types/).
	KindCounter = CounterKindBuilder{}
	// KindGauge is the starting builder instance for gauge metrics in prometheus (see also https://prometheus.io/docs/concepts/metric_types/).
	KindGauge = GaugeKindBuilder{}
	// KindHistogram is the starting builder instance for histogram metrics in prometheus (see also https://prometheus.io/docs/concepts/metric_types/).
	KindHistogram = HistogramKindBuilder{}
	// KindSummary is the starting builder instance for summary metrics in prometheus (see also https://prometheus.io/docs/concepts/metric_types/).
	KindSummary = SummaryKindBuilder{}
)

type (
	kind string
	Kind struct {
		kind kind
		help string
		// histogram metric options
		buckets []float64
		// summary metric options
		objectives map[float64]float64
		maxAge     time.Duration
		ageBuckets uint32
		bufCap     uint32
	}
	KindBuilder interface {
		Build() Kind
	}
	CounterKindBuilder struct {
		help string
	}
	GaugeKindBuilder struct {
		help string
	}
	HistogramKindBuilder struct {
		help    string
		buckets []float64
	}
	SummaryKindBuilder struct {
		help       string
		objectives map[float64]float64
		maxAge     time.Duration
		ageBuckets uint32
		bufCap     uint32
	}
)

var (
	_ KindBuilder = CounterKindBuilder{}
	_ KindBuilder = GaugeKindBuilder{}
	_ KindBuilder = HistogramKindBuilder{}
	_ KindBuilder = SummaryKindBuilder{}
)

// WithHelp attaches a help string to your metric, overwriting the default string of "Unit: <your unit>"
func (k Kind) WithHelp(help string) Kind {
	k.help = help

	return k
}

// WithHelp attaches a help string to your metric, overwriting the default string of "Unit: <your unit>"
func (k CounterKindBuilder) WithHelp(help string) CounterKindBuilder {
	k.help = help

	return k
}

// WithHelp attaches a help string to your metric, overwriting the default string of "Unit: <your unit>"
func (k GaugeKindBuilder) WithHelp(help string) GaugeKindBuilder {
	k.help = help

	return k
}

// WithHelp attaches a help string to your metric, overwriting the default string of "Unit: <your unit>"
func (k HistogramKindBuilder) WithHelp(help string) HistogramKindBuilder {
	k.help = help

	return k
}

// WithHelp attaches a help string to your metric, overwriting the default string of "Unit: <your unit>"
func (k SummaryKindBuilder) WithHelp(help string) SummaryKindBuilder {
	k.help = help

	return k
}

// Build converts your builder into a Kind you can use with a metric.
func (k CounterKindBuilder) Build() Kind {
	return Kind{
		kind: kindCounter,
		help: k.help,
	}
}

// Build converts your builder into a Kind you can use with a metric.
func (k GaugeKindBuilder) Build() Kind {
	return Kind{
		kind: kindGauge,
		help: k.help,
	}
}

// Build converts your builder into a Kind you can use with a metric.
func (k HistogramKindBuilder) Build() Kind {
	return Kind{
		kind:    kindHistogram,
		help:    k.help,
		buckets: k.buckets,
	}
}

// Build converts your builder into a Kind you can use with a metric.
func (k SummaryKindBuilder) Build() Kind {
	return Kind{
		kind:       kindSummary,
		help:       k.help,
		objectives: k.objectives,
		maxAge:     k.maxAge,
		ageBuckets: k.ageBuckets,
		bufCap:     k.bufCap,
	}
}

// WithBuckets sets the buckets your histogram will contain. From the Prometheus documentation:
//
// Buckets defines the buckets into which observations are counted. Each
// element in the slice is the upper inclusive bound of a bucket. The
// values must be sorted in strictly increasing order. There is no need
// to add a highest bucket with +Inf bound, it will be added
// implicitly. If Buckets is left as nil or set to a slice of length
// zero, it is replaced by default buckets. The default buckets are
// DefBuckets if no buckets for a native histogram (see below) are used,
// otherwise the default is no buckets. (In other words, if you want to
// use both regular buckets and buckets for a native histogram, you have
// to define the regular buckets here explicitly.)
func (k HistogramKindBuilder) WithBuckets(buckets []float64) HistogramKindBuilder {
	k.buckets = buckets

	return k
}

// WithObjectives sets the objectives your summary will contain. From the Prometheus documentation:
//
// Objectives defines the quantile rank estimates with their respective
// absolute error. If Objectives[q] = e, then the value reported for q
// will be the φ-quantile value for some φ between q-e and q+e.  The
// default value is an empty map, resulting in a summary without
// quantiles.
func (k SummaryKindBuilder) WithObjectives(objectives map[float64]float64) SummaryKindBuilder {
	k.objectives = objectives

	return k
}

// WithMaxAge sets the maximum your summary will use to compute objectives. From the Prometheus documentation:
//
// MaxAge defines the duration for which an observation stays relevant
// for the summary. Only applies to pre-calculated quantiles, does not
// apply to _sum and _count. Must be positive. The default value is
// DefMaxAge.
func (k SummaryKindBuilder) WithMaxAge(maxAge time.Duration) SummaryKindBuilder {
	k.maxAge = maxAge

	return k
}

// WithAgeBuckets sets the number of age buckets your summary will use to compute objectives. From the Prometheus documentation:
//
// AgeBuckets is the number of buckets used to exclude observations that
// are older than MaxAge from the summary. A higher number has a
// resource penalty, so only increase it if the higher resolution is
// really required. For very high observation rates, you might want to
// reduce the number of age buckets. With only one age bucket, you will
// effectively see a complete reset of the summary each time MaxAge has
// passed. The default value is DefAgeBuckets.
func (k SummaryKindBuilder) WithAgeBuckets(ageBuckets uint32) SummaryKindBuilder {
	k.ageBuckets = ageBuckets

	return k
}

// WithBufCap sets the sample stream buffer size your summary will use to compute objectives. From the Prometheus documentation:
//
// BufCap defines the default sample stream buffer size.  The default
// value of DefBufCap should suffice for most uses. If there is a need
// to increase the value, a multiple of 500 is recommended (because that
// is the internal buffer size of the underlying package
// "github.com/bmizerany/perks/quantile").
func (k SummaryKindBuilder) WithBufCap(bufCap uint32) SummaryKindBuilder {
	k.bufCap = bufCap

	return k
}
