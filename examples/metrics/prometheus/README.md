## Prometheus metrics example tutorial

### Settings

metrics writer + endpoint settings
```yml
metric:
  enabled: true
  writers: 
    - prometheus
  writer_settings:
    prometheus:
      metric_limit: 5000
      api:
        enabled: true
        port: 8092
        path: /metrics
```

### Run instructions

From the repository root, run the example. It reads `config.dist.yml` from its own directory:

```sh
cd examples/metrics/prometheus
go run .
```

The example uses only local HTTP servers; no environment variables or external services are required.
It serves the Prometheus scrape at `http://localhost:8092/metrics` and the example API on port 8088.

In a second terminal, make the requests below and scrape the metrics:

```sh
for path in current-value increase increase 1k my-summary; do
  curl --fail --silent --show-error "http://localhost:8088/${path}"
done
curl --fail --silent --show-error http://localhost:8092/metrics
```

The example API on port `:8088` exposes these endpoints:

### `GET http://localhost:8088/current-value`
Returns the current value of the counter.

### `GET http://localhost:8088/increase`
Increases the counter by one and returns the current value.

### `GET http://localhost:8088/1k`
Increases the counter by a thousand and returns the current value.

### `GET http://localhost:8088/my-summary`
Observes 1.1 times the current counter value in a summary, then increases the counter by one.

`/current-value`, `/increase` and `/1k` each write the custom counter `api_request`, using the
`handler` label values `cur`, `incr` and `1k`, respectively. `/increase` and `/1k` also add their
increments to `important_counter`. `/my-summary` observes the current value multiplied by 1.1 in
`my_summary`.

### Metric names

Metric names are authored as an optional namespace plus a leaf, then rendered according to the
writer's convention. In this example, `controllers.NewMetricController` gives custom metrics the
namespace `example`; this is explicitly set on `metric.NewWriter`, not derived from `app.namespace`
(which happens to have the same value in the config). The Prometheus writer adds its fixed `gosoline`
prefix to every metric; it does not derive a prefix from `app.namespace` or `app.name` (`prometheus`).
Dots in the namespace and leaf become underscores, and units and counter type determine the suffix:
for example, `example.api_request` is exported as
`gosoline_example_api_request_total`.

Prometheus dimension keys become labels with dots replaced by underscores: `request.method` becomes
`request_method`, while `route` and `name` remain unchanged. They are not prefixed by the metric
namespace. The complete naming, units and attributes contract is in the
[Gosoline metric specification](../../../pkg/metric/SEMCONV.md).

### Example output

After the requests above, the example's custom metrics have these series. Their values are
deterministic for that request sequence:

```text
# HELP gosoline_example_api_request_total unit: Count
# TYPE gosoline_example_api_request_total counter
gosoline_example_api_request_total{handler="1k"} 1
gosoline_example_api_request_total{handler="cur"} 1
gosoline_example_api_request_total{handler="incr"} 2
# HELP gosoline_example_important_counter_total unit: Count
# TYPE gosoline_example_important_counter_total counter
gosoline_example_important_counter_total 1002
# HELP gosoline_example_my_summary unit: Count
# TYPE gosoline_example_my_summary summary
gosoline_example_my_summary_sum 1102.2
gosoline_example_my_summary_count 1
```

The summary observes 1.1 times the counter value before `/my-summary` increments it. Its fractional
observation still has unit `Count`.

The same scrape also contains the HTTP server's built-in metrics (shown with their rendered names):

| Metric | Prometheus name | Authored unit, rendered unit and labels |
|---|---|---|
| Request duration | `gosoline_http_server_request_duration_seconds` (`_bucket`, `_sum`, `_count` series) | `UnitMillisecondsAverage`, rendered to seconds; labels `name`, `route`, `request_method`, `response_status_code` |
| Rejected requests | `gosoline_http_server_rejected_requests_total` | `Count`; labels `name`, `route`, `request_method` |
| Active requests | `gosoline_http_server_active_request_count` | `UnitCountMaximum` count gauge; label `name` |
| Open connections | `gosoline_http_server_connection_count` | `UnitCountMaximum` count gauge; label `name` |

Duration observations depend on request timing. The scrape also includes Go runtime and process
collector metrics.
