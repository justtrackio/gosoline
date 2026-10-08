# Metric Package Agent Guide

## Scope
- Collects metric data through the metric channel and flushes it via the metric daemon.
- Owns the metric emission contract: metric name formatting, dimension keys, unit representation.
- Hosts the writers exporting that contract: `cloudwatch`, `prometheus`, `otel` in-tree, plus the
  `elasticsearch` writer type whose factory is registered from outside this package.

## Key files
- `daemon.go` - `NewDaemonModule` factory, `RegisterWriterFactory`, aggregation and flush loop.
- `channel.go` - buffered channel every `Write`/`WriteOne` call feeds into.
- `writer_*.go` - backend writers selected through `metric.writers`.
- `naming.go` - the three per-writer renderers plus the semantic-convention registry.
- `contract.go` - shared public dimension keys and error-type normalization helpers; it deliberately
  contains no namespace catalog.
- `conformance_test.go` - the literal authored-name and dimension-key inventory, asserted against
  the contract without creating package import cycles or a second ownership catalog.
- `otel_naming.go` - UCUM unit and scale factor, and the OTEL non-unit derived from a plural leaf.
- `custom_units.go` - the custom aggregation units and their resolution to a base unit.
- `settings.go` - `Settings` struct read from the `metric` config key.
- `schema_version.go` - metric schema version constant, format validation, metadata registration.
- `SEMCONV.md` - the metric specification: every metric, its attributes, and the semantic convention it
  derives from. Kept up to date with every metric change.

## The emission contract
A metric is authored as a **canonical namespace plus a leaf**: lowercase, components delimited by a
dot, multiple words inside a component joined by an underscore. The canonical form carries no unit
suffix, no `_total`, no part of the application's identity, and no value that is carried as a
dimension.

Namespaces belong to the package that emits them. Each emitting package declares an unexported
`metricNamespace`-style constant **in the same const block as the metric names it owns**, never in a
file of its own; `pkg/metric` must not restore exported `metric.Namespace*` constants or
compatibility aliases. A package passes its namespace once to
`metric.NewWriter(namespace, defaults...)`, which stamps it onto every datum that does not already
carry one. A package emitting into two namespaces, such as a Kafka consumer reporting both
`kafka.consumer.*` and `kafka.*`, overrides the datum namespace explicitly.

### Naming grammar

Every metric follows these rules, so two metrics expressing the same kind of value are spelled the
same way. **No gosoline metric carries a canonical OpenTelemetry semantic-convention name**: the
conventions are followed for grammar, units and attribute shape, and the OTEL renderer prefixes every
name with `gosoline.` so nothing gosoline exports can be mistaken for the convention's metric.

1. The namespace names the owner. Never repeat it in the leaf.
2. A monotonic counter of events is `[qualifier.]<plural-noun>`, the qualifier a past participle -
   `consumed.messages`, `sent.messages`, `polls`, `rebalances`.
3. A current amount is `<thing>.count` on a gauge - `shard.count`, `active_request.count`.
4. Elapsed time is `<operation>.duration` on a histogram, in milliseconds. Never `delay`, never a
   bare `duration`.
5. Bytes are `<thing>.size`. Items per batch are `<thing>.<plural-noun>`, never `.size`.
6. A failure is not its own metric: put `error.type` on the metric recording the operation, with
   `{{default}}` when it succeeded.
7. Components are lowercase, words inside a component joined by an underscore, hierarchy by a dot.
8. An attribute key never repeats its metric's namespace. `error.type` is the one key taken verbatim
   from a semantic convention; every other key is gosoline's own and carries no prefix.

### Permanent namespace-owner table

| Namespace | Emitting package owner(s) |
|---|---|
| `blob` | `pkg/blob` |
| `cloud.aws.kinesis` | `pkg/cloud/aws/kinesis` |
| `conc.scheduler` | `pkg/conc/scheduler` |
| `db.client` | `pkg/db` |
| `db.repo` | `pkg/db-repo` |
| `ddb` | `pkg/ddb` |
| `http.client` | `pkg/http` |
| `http.server` | `pkg/httpserver` |
| `kafka` | `pkg/kafka` |
| `kafka.consumer` | `pkg/kafka/consumer` |
| `kafka.producer` | `pkg/kafka/producer` |
| `kvstore` | `pkg/kvstore` |
| `limit` | `pkg/limit` |
| `mdlsub` | `pkg/mdlsub` |
| `metric` | `pkg/metric` |
| `rpc.server` | `pkg/grpcserver` |
| `smpl` | `pkg/smpl` |
| `stream` | `pkg/stream` |

Each writer renders that one authored name into its own convention:

| Writer | Rendering of `http.server` + `request.duration` |
|--------|--------------------------------------------------|
| CloudWatch | `HttpServerRequestDuration`, unscaled, milliseconds |
| Prometheus | `gosoline_http_server_request_duration_seconds`, scaled to seconds |
| OTEL | `gosoline.http.server.request.duration`, unit `s` on the instrument |

The OTEL renderer prefixes `gosoline.` unconditionally, so no exported name is a canonical
semantic-convention metric even where the namespace and leaf coincide with one.
The Prometheus writer carries the fixed namespace `gosoline`, which names the framework that authored
the metric: application identity belongs in the labels the scrape target is discovered with, so it is
not part of the exported name. Prometheus adds the base-unit suffix and `_total` on counters, both
from `prometheusUnitSuffix`. Neither suffix nor prefix exists in the authored name. Prometheus also
renders dimension keys, replacing dots with underscores, because a dot is not a valid label name
character there and a datum carrying one is rejected at registration.

A datum without a namespace - one authored outside gosoline - is rendered by exactly the same rules,
minus the namespace: CloudWatch PascalCases its leaf and Prometheus replaces its dots and appends the
convention suffixes. A leaf carries canonical separators whether or not a namespace precedes it, so
the renderers may not pass it through untouched.

### Help texts
Every authored metric has a help text, registered once by its emitting package through
`metric.RegisterHelp(namespace, metricName, help)` next to the name it describes. The Prometheus and
OTEL writers resolve it via `resolveHelp`, which prefers a help the datum's own `Kind` carries (set
with `WithHelp`), falls back to the registered one, and only then to a description of the unit.

Registering the same help twice is a no-op. Registering a **different** help for the same name panics
on purpose: a backend keeps one description per metric name, so a second description is rejected when
the metric is registered and that metric silently stops being exported. Each metric now has exactly
one emitting package, so no help text needs to be shared across packages.

Every gosoline metric declares its `Kind` explicitly. Unit-based inference (`inferKind`) remains only
as the fallback for metrics authored outside gosoline, and is shared by both writers so they can never
classify one datum differently.

Adding or changing a metric means updating `authoredNames` in `conformance_test.go`, registering a
help text for it in the emitting package, **and updating `SEMCONV.md` in the same commit**; the
conformance test fails the build on a name that violates the contract, on a duplicate, and on a
rendering regression, but it cannot check the specification, so that part is on you. Removing the last
emitter of a metric means removing its `authoredNames` entry and its `SEMCONV.md` row in the same
change, so neither ever claims a metric nothing writes.

`SEMCONV.md` is the specification: it records every metric with its unit, instrument type, attributes
and the semantic-convention metric it derives from or deliberately diverges from, plus the exceptions to
the grammar and why each exists.

### Dimension-key policy
OpenTelemetry semantic-convention attributes may be added where a convention defines the relevant
attribute. Existing custom dimensions are unchanged in this focused revision. For future custom
attributes, use a unique owned prefix such as `gosoline.*`; do not retroactively rename existing
custom keys solely to apply that convention.

## Metric schema version
The metric schema version identifies the metric emission contract a gosoline build implements, so
tooling (dashboard generators, alert provisioning, metric pipelines) can branch on it without
inspecting the gosoline version.

- Current value: `v2.0`, defined by the exported constant `metric.SchemaVersion` in
  `schema_version.go`. That constant is the single source of truth - no other package may define
  the literal value.
- `v2.0` is the **one** MAJOR increment covering the whole migration off the `v1.0` contract. Every
  further rename, removal or unit change made while that migration is unmerged belongs to the same
  increment: do not raise the version again for one of them, and never publish above `v2.0` from
  this revision.
- Metadata key: `metric.schema_version` (`metric.MetadataKeySchemaVersion`). The value is written
  into the `appctx.Metadata` carrier and therefore served by the metadata server's root route.
- Format: `v<MAJOR>.<MINOR>`, both components decimal integers of 1 to 9 digits without leading
  zeros unless the component is exactly `0`. `metric.IsValidSchemaVersion` enforces it.

### Increment rules for future revisions
- **MAJOR**: a metric name, dimension key, or unit representation is removed or renamed. Increment
  MAJOR by 1 and reset MINOR to 0 - this also applies when the same change adds something.
- **MINOR**: the change is purely additive (new metric name, dimension key, or unit
  representation, nothing removed or renamed). Increment MINOR by 1, leave MAJOR unchanged.
- **unchanged**: every metric name, dimension key, and unit representation stays as it is. Leave
  the version untouched, including for refactorings and performance work.

### Release process
1. For a future contract change, bump `metric.SchemaVersion` in `schema_version.go` in the same
   commit, following the increment rules above unless maintainers explicitly direct an exception.
2. Update the current value in this file so the documented value never drifts from the constant.
3. Describe the contract change and the version decision in `RELEASE_NOTES.md`.

### Publication requirements
The version is published only by an **enabled metric daemon**: the application must wire
`application.WithMetrics` and be configured with `metric.enabled: true`. `NewDaemonModule`
registers the version right after the enabled check, during kernel build, so the entry is present
before the metadata server serves its first response. With `metric.enabled: false`, or without
`application.WithMetrics`, the metadata document contains no `metric` member at all - the absence
is the intended signal that the application emits no metrics.

## Common tasks
- Add a writer: implement `Writer` and register it through `RegisterWriterFactory`, then document
  its settings key below `metric.writer_settings`.
- Add or change an emitted metric: declare the namespace in the emitting package, update the literal
  `authoredNames` inventory, and apply the future increment rules unless maintainers direct an
  exception.
- Adjust default metrics: see `defaults.go` and the per-package `metric.Datum` producers.

## Testing
- `go test ./pkg/metric/...` before pushing changes.

## Required config keys
```yaml
metric:
  enabled: true               # Enables the metric daemon; required for schema version publication
  interval: 60s               # Flush interval
  writers:                    # Backends; any subset of the supported writers
    - cloudwatch
```

## Related packages
- `pkg/appctx` - metadata carrier the schema version is registered in
- `pkg/application` - `WithMetrics` wiring and the metadata server exposing the version
- `pkg/kernel` - module lifecycle; the daemon factory runs during kernel build
- `pkg/otel` - shared OTLP exporter configuration used by the `otel` writer

## Tips
- Never derive the reported schema version from configuration - report the constant verbatim.
- Registration performs exactly one attempt and returns a wrapped error; `NewDaemonModule` returns
  that error unchanged, so the kernel build aborts instead of shipping an incomplete metadata
  document.
