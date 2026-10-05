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

run `go run main.go`

The application exposes the prometheus metrics on `http://localhost:8092/metrics`
Additionally there is a basic api server running on port `:8088` with the following endpoints

### `GET http://localhost:8088/current-value`
returns the current value of the counter
### `GET http://localhost:8088/increase`
increases the counter by one and returns the current value
### `GET http://localhost:8088/1k`
increases the counter by a thousand and returns the current value
### `GET http://localhost:8088/my-summary`
observes the current value in a summary, increases the counter by one and returns the current value

`/current-value`, `/increase` and `/1k` each write the custom counter `api_request`, using the handler
name as a label. `/increase` and `/1k` additionally add their increment to `important_counter`, and
`/my-summary` observes into the summary `my_summary`.

### Metric names

An application authors its own metric names and gosoline exports them unchanged - `api_request` stays
`api_request`. Every metric gosoline itself emits is authored as a canonical namespace plus a leaf and
rendered into the prometheus convention: the namespace becomes the subsystem, dots become underscores,
and the name carries a base-unit suffix plus `_total` on counters. So the HTTP server's request
duration is exported as `<prefix>_http_server_request_duration_seconds`, in seconds. The same rule
applies to dimensions, which become labels with dots replaced by underscores (`http.route` becomes
`http_route`). See `pkg/metric/AGENTS.md` for the full contract.

The `<prefix>` above is the application-derived prometheus namespace, `{app.namespace}-{app.name}` by
default, so with this example's config it is `example_prometheus`.

### Example output

Requesting `/current-value` once, `/increase` twice, `/1k` once and `/my-summary` once, then scraping
the `/metrics` endpoint. Go runtime and process collectors are omitted, as are the histogram's
`_bucket` series.

```
# HELP example_prometheus_api_request unit: Count
# TYPE example_prometheus_api_request counter
example_prometheus_api_request{handler="1k"} 1
example_prometheus_api_request{handler="cur"} 1
example_prometheus_api_request{handler="incr"} 2
# HELP example_prometheus_important_counter unit: Count
# TYPE example_prometheus_important_counter counter
example_prometheus_important_counter 1002
# HELP example_prometheus_my_summary unit: 
# TYPE example_prometheus_my_summary summary
example_prometheus_my_summary_sum 1102.2
example_prometheus_my_summary_count 1
# HELP example_prometheus_http_server_request_duration_seconds unit: UnitMillisecondsAverage
# TYPE example_prometheus_http_server_request_duration_seconds histogram
example_prometheus_http_server_request_duration_seconds_sum{http_request_method="GET",http_response_status_code="200",http_route="/1k",http_server_name="default"} 2.8625e-05
example_prometheus_http_server_request_duration_seconds_count{http_request_method="GET",http_response_status_code="200",http_route="/1k",http_server_name="default"} 1
example_prometheus_http_server_request_duration_seconds_sum{http_request_method="GET",http_response_status_code="200",http_route="/current-value",http_server_name="default"} 0.000405042
example_prometheus_http_server_request_duration_seconds_count{http_request_method="GET",http_response_status_code="200",http_route="/current-value",http_server_name="default"} 1
example_prometheus_http_server_request_duration_seconds_sum{http_request_method="GET",http_response_status_code="200",http_route="/increase",http_server_name="default"} 8.8126e-05
example_prometheus_http_server_request_duration_seconds_count{http_request_method="GET",http_response_status_code="200",http_route="/increase",http_server_name="default"} 2
# HELP example_prometheus_http_server_rejected_requests_total unit: Count
# TYPE example_prometheus_http_server_rejected_requests_total counter
example_prometheus_http_server_rejected_requests_total{http_request_method="GET",http_route="/1k",http_server_name="default"} 0
example_prometheus_http_server_rejected_requests_total{http_request_method="GET",http_route="/current-value",http_server_name="default"} 0
example_prometheus_http_server_rejected_requests_total{http_request_method="GET",http_route="/increase",http_server_name="default"} 0
example_prometheus_http_server_rejected_requests_total{http_request_method="GET",http_route="/my-summary",http_server_name="default"} 0
# HELP example_prometheus_http_server_active_requests unit: UnitCountMaximum
# TYPE example_prometheus_http_server_active_requests gauge
example_prometheus_http_server_active_requests{http_server_name="default"} 0
# HELP example_prometheus_http_server_connection_count unit: UnitCountMaximum
# TYPE example_prometheus_http_server_connection_count gauge
example_prometheus_http_server_connection_count{http_server_name="default"} 0
```

There is no separate request counter: the number of requests is the observation count of the duration
histogram, which is the `_count` series above.

Note that `my_summary` reports an empty unit and makes the metric daemon log
`invalid metric: metric my_summary has no unit`, because this example writes it without one. An
application metric should always carry a unit.
