package metric

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// authoredName is one entry of the metric emission contract: the canonical namespace and leaf a
// gosoline metric is authored as, together with the base unit it measures in, the instrument type it
// declares, and whether an OpenTelemetry semantic convention owns it.
type authoredName struct {
	namespace string
	leaf      string
	unit      types.StandardUnit
	kind      kind
}

// authoredNames is the full set of metrics gosoline emits. Every emitting package authors its metrics
// from this contract, and the conformance tests below assert that each entry satisfies the canonical
// form and renders correctly for CloudWatch, Prometheus and OTEL. A metric gosoline emits that is
// missing here, or an entry no package emits, is a contract defect.
var authoredNames = []authoredName{
	{"stream", "process.duration", UnitMilliseconds, kindHistogram},
	{"stream", "consumed.messages", UnitCount, kindCounter},
	{"stream", "errors", UnitCount, kindCounter},
	{"stream", "retry.operations", UnitCount, kindCounter},
	{"stream", "produced.messages", UnitCount, kindCounter},
	{"stream", "batch.messages", UnitCount, kindHistogram},
	{"stream", "aggregate.messages", UnitCount, kindHistogram},
	{"stream", "idle.duration", UnitMilliseconds, kindHistogram},
	{"stream", "message.count", UnitCount, kindGauge},
	{"stream", "reads", UnitCount, kindCounter},
	{"stream", "writes", UnitCount, kindCounter},

	{"kafka.consumer", "consumed.messages", UnitCount, kindCounter},
	{"kafka.consumer", "process.duration", UnitMilliseconds, kindHistogram},
	{"kafka.consumer", "polls", UnitCount, kindCounter},
	{"kafka.consumer", "poll.duration", UnitMilliseconds, kindHistogram},
	{"kafka.consumer", "commit.duration", UnitMilliseconds, kindHistogram},
	{"kafka.consumer", "commit.errors", UnitCount, kindCounter},
	{"kafka.consumer", "wait.duration", UnitMilliseconds, kindHistogram},
	{"kafka.consumer", "rebalances", UnitCount, kindCounter},
	{"kafka.consumer", "consume.errors", UnitCount, kindCounter},
	{"kafka.producer", "sent.messages", UnitCount, kindCounter},
	{"kafka.producer", "produce.duration", UnitMilliseconds, kindHistogram},
	{"kafka.producer", "batch.records", UnitCount, kindHistogram},
	{"kafka.producer", "send.errors", UnitCount, kindCounter},

	{"kafka", "connects", UnitCount, kindCounter},
	{"kafka", "throttles", UnitCount, kindCounter},
	{"kafka", "throttle.duration", UnitMilliseconds, kindHistogram},
	{"kafka", "produce.batch.records", UnitCount, kindHistogram},
	{"kafka", "produce.batch.size", UnitBytes, kindHistogram},
	{"kafka", "produce.batch.compressed.size", UnitBytes, kindHistogram},
	{"kafka", "fetch.batch.records", UnitCount, kindHistogram},
	{"kafka", "fetch.batch.size", UnitBytes, kindHistogram},
	{"kafka", "fetch.batch.compressed.size", UnitBytes, kindHistogram},

	{"cloud.aws.kinesis", "consumed.messages", UnitCount, kindCounter},
	{"cloud.aws.kinesis", "sent.messages", UnitCount, kindCounter},
	{"cloud.aws.kinesis", "process.duration", UnitMilliseconds, kindHistogram},
	{"cloud.aws.kinesis", "reads", UnitCount, kindCounter},
	{"cloud.aws.kinesis", "consume.errors", UnitCount, kindCounter},
	{"cloud.aws.kinesis", "lag", UnitMilliseconds, kindGauge},
	{"cloud.aws.kinesis", "acquire.duration", UnitSeconds, kindHistogram},
	{"cloud.aws.kinesis", "sleep.duration", UnitMilliseconds, kindHistogram},
	{"cloud.aws.kinesis", "wait.duration", UnitMilliseconds, kindHistogram},
	{"cloud.aws.kinesis", "shard.count", UnitCount, kindGauge},
	{"cloud.aws.kinesis", "client.count", UnitCount, kindGauge},
	{"cloud.aws.kinesis", "batch.records", UnitCount, kindHistogram},
	{"cloud.aws.kinesis", "send.errors", UnitCount, kindCounter},

	{"kvstore", "reads", UnitCount, kindCounter},
	{"kvstore", "writes", UnitCount, kindCounter},
	{"kvstore", "deletes", UnitCount, kindCounter},
	{"kvstore", "hits", UnitCount, kindCounter},
	{"kvstore", "item.count", UnitCount, kindGauge},

	{"db.client", "connection.count", UnitCount, kindGauge},
	{"db.client", "connections", UnitCount, kindCounter},
	{"db.repo", "operation.duration", UnitMilliseconds, kindHistogram},
	{"db.repo", "model_event.notifications", UnitCount, kindCounter},
	{"ddb", "operation.duration", UnitMilliseconds, kindHistogram},

	{"mdlsub", "consumed.events", UnitCount, kindCounter},
	{"mdlsub", "skipped.events", UnitCount, kindCounter},
	{"mdlsub", "consume.errors", UnitCount, kindCounter},

	{"http.server", "request.duration", UnitMilliseconds, kindHistogram},
	{"http.server", "rejected.requests", UnitCount, kindCounter},
	{"http.server", "active_request.count", UnitCount, kindGauge},
	{"http.server", "connection.count", UnitCount, kindGauge},
	{"http.client", "request.duration", UnitMilliseconds, kindHistogram},
	{"rpc.server", "request.duration", UnitMilliseconds, kindHistogram},

	{"blob", "batch.operations", UnitCount, kindCounter},
	{"conc.scheduler", "batch.tasks", UnitCount, kindHistogram},
	{"conc.scheduler", "task.queue.duration", UnitMilliseconds, kindHistogram},
	{"limit", "takes", UnitCount, kindCounter},
	{"limit", "releases", UnitCount, kindCounter},
	{"limit", "throttles", UnitCount, kindCounter},
	{"limit", "errors", UnitCount, kindCounter},
	{"smpl", "decisions", UnitCount, kindCounter},
	{"metric", "log.records", UnitCount, kindCounter},
}

// canonicalComponent matches one component of a canonical name: lowercase, starting with a letter,
// with multiple words joined by a single underscore.
var canonicalComponent = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)*$`)

// unitSuffixes are the words a canonical name must not end in, because a unit belongs on the
// instrument rather than in the name.
var unitSuffixes = []string{
	"seconds", "second", "milliseconds", "millisecond", "ms", "microseconds", "microsecond", "us",
	"nanoseconds", "nanosecond", "ns", "bytes", "byte", "kb", "mb", "gb", "tb", "bits", "bit",
	"percent", "ratio", "total",
}

// identityWords are the words a canonical name must not contain, because identity is carried as
// resource attributes and backend namespaces instead.
var identityWords = []string{
	"env", "environment", "app", "application", "team", "project", "family", "organization", "org",
	"instance",
}

// compoundWords are the multi-word terms the contract uses. Each is one component with its words
// joined by an underscore, so finding one split across two dot-delimited components means a word
// boundary was mistaken for a component boundary.
var compoundWords = []string{
	"active_requests", "model_event", "rate_limit", "redis_list", "status_code",
}

// TestAuthoredNamesAreCanonical asserts the canonical form of every authored namespace and leaf, so a
// name violating the contract fails the build rather than review.
func TestAuthoredNamesAreCanonical(t *testing.T) {
	for _, name := range authoredNames {
		t.Run(canonicalName(name.namespace, name.leaf), func(t *testing.T) {
			assert.NoError(t, checkCanonicalForm(name))
		})
	}
}

// TestAuthoredNamesAreUnique asserts no canonical name is authored twice, which would let two metrics
// collide on one instrument.
func TestAuthoredNamesAreUnique(t *testing.T) {
	seen := make(map[string]authoredName, len(authoredNames))

	for _, name := range authoredNames {
		full := canonicalName(name.namespace, name.leaf)

		previous, duplicate := seen[full]
		require.False(t, duplicate, "%s is authored twice, as %v and as %v", full, previous, name)

		seen[full] = name
	}
}

// TestCanonicalFormCheckRejectsInvalidNames pins down that the conformance check fails on the
// violations it exists to catch, so a green conformance run means something.
func TestCanonicalFormCheckRejectsInvalidNames(t *testing.T) {
	tests := map[string]authoredName{
		"uppercase leaf": {
			namespace: "stream.consumer", leaf: "Errors", unit: UnitCount, kind: kindCounter,
		},
		"uppercase namespace": {
			namespace: "StreamConsumer", leaf: "errors", unit: UnitCount, kind: kindCounter,
		},
		"word boundary split into components": {
			namespace: "http.server", leaf: "response.status.code", unit: UnitCount, kind: kindGauge,
		},
		"unit suffix on the leaf": {
			namespace: "aws.kinesis.shard", leaf: "milliseconds_behind", unit: UnitMilliseconds, kind: kindGauge,
		},
		"unit suffix as its own component": {
			namespace: "kafka.consumer", leaf: "delay.seconds", unit: UnitSeconds, kind: kindHistogram,
		},
		"total suffix": {
			namespace: "stream.consumer", leaf: "errors.total", unit: UnitCount, kind: kindCounter,
		},
		"identity in the name": {
			namespace: "stream.consumer", leaf: "application.errors", unit: UnitCount, kind: kindCounter,
		},
		"plural leaf measuring a unit": {
			namespace: "kafka.broker", leaf: "throttle.durations", unit: UnitMilliseconds, kind: kindHistogram,
		},
		"hyphen in the leaf": {
			namespace: "stream.consumer", leaf: "retry-operations", unit: UnitCount, kind: kindCounter,
		},
		"empty leaf": {
			namespace: "stream.consumer", leaf: "", unit: UnitCount, kind: kindCounter,
		},
		"empty namespace": {
			namespace: "", leaf: "errors", unit: UnitCount, kind: kindCounter,
		},
		"double underscore": {
			namespace: "stream.consumer", leaf: "retry__operations", unit: UnitCount, kind: kindCounter,
		},
	}

	for name, invalid := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Error(t, checkCanonicalForm(invalid))
		})
	}
}

// TestAuthoredNamesRenderForEveryWriter asserts the three renderings of every authored name, so a
// change to a renderer that breaks an exported name fails the build.
func TestAuthoredNamesRenderForEveryWriter(t *testing.T) {
	for _, name := range authoredNames {
		t.Run(canonicalName(name.namespace, name.leaf), func(t *testing.T) {
			cloudWatch := renderCloudWatchName(name.namespace, name.leaf)
			assert.Regexp(t, `^[A-Z][A-Za-z0-9]*$`, cloudWatch, "cloudwatch name must be PascalCase")
			assert.NotContains(t, cloudWatch, "Gosoline", "cloudwatch carries no gosoline prefix")

			subsystem, promName := renderPrometheusName(name.namespace, name.leaf, name.unit, name.kind)
			assert.Regexp(t, `^[a-z][a-z0-9_]*$`, subsystem, "prometheus subsystem must be snake_case")
			assert.Regexp(t, `^[a-z][a-z0-9_]*$`, promName, "prometheus name must be snake_case")
			assert.Equal(t, name.kind == kindCounter, strings.HasSuffix(promName, "_total"),
				"only a counter carries the _total suffix")
			assert.Equal(t, prometheusUnitSuffix(name.unit, kindGauge) != "",
				strings.Contains(promName, "_seconds") || strings.Contains(promName, "_bytes"),
				"a measured unit carries a base-unit suffix")

			otelName := renderOtelName(name.namespace, name.leaf)
			assert.True(t, strings.HasPrefix(otelName, otelGosolinePrefix),
				"every gosoline metric is rendered with the gosoline prefix")
			assert.Equal(t, canonicalName(name.namespace, name.leaf), strings.TrimPrefix(otelName, otelGosolinePrefix),
				"otel renders the canonical dotted form")
			assert.NotContains(t, otelName, "_total", "otel carries no total suffix")
			assert.NotContains(t, otelName, "_seconds", "otel carries no unit suffix")
		})
	}
}

// TestAuthoredNamesDeriveTheirOtelUnit asserts the OTEL instrument unit of every authored name,
// including the non-unit derived from a plural leaf, so a singularization surprise fails the build
// rather than showing up in a backend.
func TestAuthoredNamesDeriveTheirOtelUnit(t *testing.T) {
	expected := map[string]string{
		"stream.process.duration":   "s",
		"stream.consumed.messages":  "{message}",
		"stream.errors":             "{error}",
		"stream.retry.operations":   "{operation}",
		"stream.produced.messages":  "{message}",
		"stream.batch.messages":     "{message}",
		"stream.aggregate.messages": "{message}",
		"stream.idle.duration":      "s",
		"stream.message.count":      "1",
		"stream.reads":              "{read}",
		"stream.writes":             "{write}",

		"kafka.consumer.consumed.messages": "{message}",
		"kafka.consumer.process.duration":  "s",
		"kafka.consumer.polls":             "{poll}",
		"kafka.consumer.poll.duration":     "s",
		"kafka.consumer.commit.duration":   "s",
		"kafka.consumer.commit.errors":     "{error}",
		"kafka.consumer.wait.duration":     "s",
		"kafka.consumer.rebalances":        "{rebalance}",
		"kafka.consumer.consume.errors":    "{error}",
		"kafka.producer.sent.messages":     "{message}",
		"kafka.producer.produce.duration":  "s",
		"kafka.producer.batch.records":     "{record}",
		"kafka.producer.send.errors":       "{error}",

		"kafka.connects":                      "{connect}",
		"kafka.throttles":                     "{throttle}",
		"kafka.throttle.duration":             "s",
		"kafka.produce.batch.records":         "{record}",
		"kafka.produce.batch.size":            "By",
		"kafka.produce.batch.compressed.size": "By",
		"kafka.fetch.batch.records":           "{record}",
		"kafka.fetch.batch.size":              "By",
		"kafka.fetch.batch.compressed.size":   "By",

		"cloud.aws.kinesis.consumed.messages": "{message}",
		"cloud.aws.kinesis.sent.messages":     "{message}",
		"cloud.aws.kinesis.process.duration":  "s",
		"cloud.aws.kinesis.reads":             "{read}",
		"cloud.aws.kinesis.consume.errors":    "{error}",
		"cloud.aws.kinesis.lag":               "s",
		"cloud.aws.kinesis.acquire.duration":  "s",
		"cloud.aws.kinesis.sleep.duration":    "s",
		"cloud.aws.kinesis.wait.duration":     "s",
		"cloud.aws.kinesis.shard.count":       "1",
		"cloud.aws.kinesis.client.count":      "1",
		"cloud.aws.kinesis.batch.records":     "{record}",
		"cloud.aws.kinesis.send.errors":       "{error}",

		"kvstore.reads":      "{read}",
		"kvstore.writes":     "{write}",
		"kvstore.deletes":    "{delete}",
		"kvstore.hits":       "{hit}",
		"kvstore.item.count": "1",

		"db.client.connection.count":        "1",
		"db.client.connections":             "{connection}",
		"db.repo.operation.duration":        "s",
		"db.repo.model_event.notifications": "{notification}",
		"ddb.operation.duration":            "s",

		"mdlsub.consumed.events": "{event}",
		"mdlsub.skipped.events":  "{event}",
		"mdlsub.consume.errors":  "{error}",

		"http.server.request.duration":     "s",
		"http.server.rejected.requests":    "{request}",
		"http.server.active_request.count": "1",
		"http.server.connection.count":     "1",
		"http.client.request.duration":     "s",
		"rpc.server.request.duration":      "s",

		"blob.batch.operations":              "{operation}",
		"conc.scheduler.batch.tasks":         "{task}",
		"conc.scheduler.task.queue.duration": "s",
		"limit.takes":                        "{take}",
		"limit.releases":                     "{release}",
		"limit.throttles":                    "{throttle}",
		"limit.errors":                       "{error}",
		"smpl.decisions":                     "{decision}",
		"metric.log.records":                 "{record}",
	}

	for _, name := range authoredNames {
		full := canonicalName(name.namespace, name.leaf)

		t.Run(full, func(t *testing.T) {
			want, ok := expected[full]
			require.True(t, ok, "%s has no expected otel unit", full)
			assert.Equal(t, want, otelInstrumentUnit(name.leaf, name.unit))
		})
	}
}

// checkCanonicalForm reports every way an authored name violates the canonical form.
func checkCanonicalForm(name authoredName) error {
	if name.namespace == "" {
		return fmt.Errorf("namespace is empty")
	}

	if name.leaf == "" {
		return fmt.Errorf("leaf is empty")
	}

	if err := checkComponents("namespace", name.namespace); err != nil {
		return err
	}

	if err := checkComponents("leaf", name.leaf); err != nil {
		return err
	}

	full := canonicalName(name.namespace, name.leaf)

	if err := checkCompoundWordsAreOneComponent(full); err != nil {
		return err
	}

	if err := checkWordsCarryNoUnitOrIdentity(full); err != nil {
		return err
	}

	return checkPluralOnlyCountsThings(full, name.unit)
}

// checkComponents reports a component that is not lowercase snake_case.
func checkComponents(label string, part string) error {
	for _, component := range strings.Split(part, ".") {
		if !canonicalComponent.MatchString(component) {
			return fmt.Errorf("%s component %q is not lowercase snake_case", label, component)
		}
	}

	return nil
}

// checkCompoundWordsAreOneComponent reports a multi-word term whose word boundary was mistaken for a
// component boundary.
func checkCompoundWordsAreOneComponent(full string) error {
	for _, compound := range compoundWords {
		if strings.Contains(full, strings.ReplaceAll(compound, "_", ".")) {
			return fmt.Errorf("%s splits the term %q across two components instead of joining its words", full, compound)
		}
	}

	return nil
}

// checkWordsCarryNoUnitOrIdentity reports a name carrying a unit, a total marker or part of the
// application's identity, all of which belong outside the name.
func checkWordsCarryNoUnitOrIdentity(full string) error {
	for _, word := range strings.FieldsFunc(full, isCanonicalSeparator) {
		if slices.Contains(unitSuffixes, word) {
			return fmt.Errorf("%s carries the unit or total word %q", full, word)
		}

		if slices.Contains(identityWords, word) {
			return fmt.Errorf("%s carries the identity word %q", full, word)
		}
	}

	return nil
}

// checkPluralOnlyCountsThings reports a plural leaf that measures a unit, because a plural name states
// that the metric counts discrete things.
func checkPluralOnlyCountsThings(full string, unit types.StandardUnit) error {
	if _, plural := singularize(lastComponent(full)); plural && resolveBaseUnit(unit) != UnitCount {
		return fmt.Errorf("%s is plural but measures %s rather than counting discrete things", full, unit)
	}

	return nil
}

// authoredDimensionKeys is the set of distinct dimension keys gosoline emits, together with whether an
// OpenTelemetry semantic convention defines the key. A key gosoline invents must be canonical; a key a
// convention defines is spelled the way the convention spells it.
var authoredDimensionKeys = map[string]bool{
	// the one key gosoline takes verbatim from a semantic convention
	"error.type": true,

	// keys gosoline authors itself. None carries a namespace prefix: the metric it is attached to
	// already names its subsystem.
	"broker.address":       false,
	"client.name":          false,
	"client.type":          false,
	"connection.state":     false,
	"consumer.name":        false,
	"level":                false,
	"list.name":            false,
	"method":               false,
	"model.id":             false,
	"name":                 false,
	"operation":            false,
	"operation.name":       false,
	"partition.id":         false,
	"prefix":               false,
	"producer.name":        false,
	"request.method":       false,
	"response.status_code": false,
	"route":                false,
	"sampled":              false,
	"service":              false,
	"store.type":           false,
	"stream.name":          false,
	"topic.name":           false,
}

// TestAuthoredDimensionKeysAreCanonical asserts every dimension key gosoline emits is authored in the
// same canonical form metric names are, so a key can not drift into PascalCase or into a word boundary
// mistaken for a component boundary.
func TestAuthoredDimensionKeysAreCanonical(t *testing.T) {
	for key := range authoredDimensionKeys {
		t.Run(key, func(t *testing.T) {
			for _, component := range strings.Split(key, ".") {
				assert.Regexp(t, canonicalComponent, component, "component %q of %q is not lowercase snake_case", component, key)
			}

			for _, compound := range compoundWords {
				assert.NotContains(t, key, strings.ReplaceAll(compound, "_", "."),
					"%q splits the term %q across two components", key, compound)
			}
		})
	}
}

// TestAuthoredDimensionKeysRenderForPrometheus asserts every dimension key renders into a valid
// Prometheus label name. A dot is not a valid character there, and a datum carrying one is rejected at
// registration, which drops the metric silently.
func TestAuthoredDimensionKeysRenderForPrometheus(t *testing.T) {
	promLabelName := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

	for key := range authoredDimensionKeys {
		t.Run(key, func(t *testing.T) {
			assert.Regexp(t, promLabelName, renderPrometheusLabelName(key))
		})
	}
}

// TestSharedDimensionKeysAreAuthored asserts the keys this package publishes for every other package to
// use are part of the authored set, so the set can not fall behind the constants.
func TestSharedDimensionKeysAreAuthored(t *testing.T) {
	shared := []string{
		DimensionErrorType,
		DimensionLogLevel,
		DimensionModelId,
	}

	for _, key := range shared {
		_, ok := authoredDimensionKeys[key]
		assert.True(t, ok, "%s is published but not part of the authored dimension key set", key)
	}
}
