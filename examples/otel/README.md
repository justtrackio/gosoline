# Gosoline OpenTelemetry metric demo

This example runs a long-lived gosoline application that emits three metric Datum kinds through the native OTEL metric writer:

- `demo_counter` — `Counter`, in counts
- `demo_gauge` — `Gauge`, in counts
- `demo_histogram` — `Histogram`, in milliseconds

The application emits one batch immediately and then emits another batch every 10 seconds. Each metric has a distinct value, and each value changes on every emission. The application also writes one structured log record per metric so the value sent to the metric writer is visible.

A pinned OpenTelemetry Collector runs in Docker. It receives OTLP/gRPC metrics on port `4317` and prints detailed metric data to stdout with the Collector `debug` exporter.

## Prerequisites

- Go 1.24 or newer
- Docker Desktop or another running Docker daemon
- Port `4317` available on localhost

## Run the application and Collector together

From this directory, run:

```shell
./run.sh
```

`run.sh` starts `otel/opentelemetry-collector:0.154.0`, mounts `collector-config.yaml`, publishes the Collector's OTLP/gRPC port, and then runs the gosoline application with `config.yaml`. Both streams are shown live in the same terminal with `[collector]` and `[application]` prefixes. Press `Ctrl-C` to stop both processes.

The image can be overridden when testing another Collector build:

```shell
OTEL_COLLECTOR_IMAGE=otel/opentelemetry-collector:0.154.0 ./run.sh
```

## Run the processes separately

Start the Collector in one terminal:

```shell
docker run --rm \
  --publish 4317:4317 \
  --volume "$PWD/collector-config.yaml:/etc/otelcol/config.yaml:ro" \
  otel/opentelemetry-collector:0.154.0 \
  --config=/etc/otelcol/config.yaml
```

Then start the gosoline application in another terminal:

```shell
go run .
```

## Emitted values

For emission number `n`, the application sends the following Datum values:

| Metric | Datum kind | Unit | Value |
| --- | --- | --- | --- |
| `demo_counter` | `Counter` | count | `n` |
| `demo_gauge` | `Gauge` | count | `100 + 10*n` |
| `demo_histogram` | `Histogram` | milliseconds | `500 + 100*n` |

For example, emission 1 sends `1`, `110`, and `600`; emission 2 sends `2`, `120`, and `700`.

The application logs records containing `emission`, `kind`, `metric`, `unit`, and `value` fields alongside the message `emitted metric`. The Collector output contains the corresponding OTEL metric names and data types under `ResourceMetrics`.

Because OTEL counters are additive, the Collector reports the accumulated counter sum rather than each raw counter increment. Gauges report their latest value, while histograms report the observations and their aggregate statistics. This is expected OTEL behavior; compare the application log's raw Datum value with the Collector's data type and aggregate output.

The application has no HTTP route and remains running until its context is cancelled. `Ctrl-C` lets gosoline shut down the metric daemon and OTEL exporter cleanly.

## Configuration files

- `config.yaml` is the local runnable configuration. It sends metrics to `localhost:4317`, uses the OTEL writer without gosoline-side aggregation, and exports every 10 seconds.
- `collector-config.yaml` configures the OTLP/gRPC receiver and detailed `debug` exporter output.
- `config.dist.yml` remains the broader OTEL configuration used by the repository's OTEL integration test setup.
