## Prometheus metrics example tutorial

### Settings

metrics endpoint settings
```yml
prometheus:
  metric_limit: 5000
  api:
    enabled: true
    port: 8092
    path: /metrics
```

metrics writter settings
```yml
metric:
    enabled: true
    interval: 60s
    writer: prom
```


### Run instructions

run `go run main.go`

The application exposes the prometheus metrics on `http://localhost:8092/metrics`
Additionally there is a basic api server running on port `:8088` with the following endpoints

### `GET http://localhost:8088/current-value`
return the current value of the counter
### `GET http://localhost:8088/increase`
increases the counter by one and returns the current value
### `GET http://localhost:8088/decrease`
decreases the counter by one and returns the current value

all the above endpoints also have a custom metric `api_request` which increments
every time an endpoint is called (the handler name is used as a label)


example output of custom metrics when requesting the `/metrics` endpoint
```
# HELP Gosoline:dev:metrics:prometheus_ApiRequestCount unit: Count
# TYPE Gosoline:dev:metrics:prometheus_ApiRequestCount counter
gosoline:dev:metrics:prometheus_ApiRequestCount{path="/current-value"} 4
gosoline:dev:metrics:prometheus_ApiRequestCount{path="/decrease"} 4
gosoline:dev:metrics:prometheus_ApiRequestCount{path="/increase"} 5
# HELP Gosoline:dev:metrics:prometheus_ApiRequestResponseTime unit: UnitMillisecondsAverage
# TYPE Gosoline:dev:metrics:prometheus_ApiRequestResponseTime gauge
gosoline:dev:metrics:prometheus_ApiRequestResponseTime{path="/current-value"} 0.042044
gosoline:dev:metrics:prometheus_ApiRequestResponseTime{path="/decrease"} 0.040718
gosoline:dev:metrics:prometheus_ApiRequestResponseTime{path="/increase"} 0.044629
# HELP Gosoline:dev:metrics:prometheus_ApiStatus2XX unit: Count
# TYPE Gosoline:dev:metrics:prometheus_ApiStatus2XX counter
gosoline:dev:metrics:prometheus_ApiStatus2XX{path="/current-value"} 4
gosoline:dev:metrics:prometheus_ApiStatus2XX{path="/decrease"} 4
gosoline:dev:metrics:prometheus_ApiStatus2XX{path="/increase"} 5
# HELP Gosoline:dev:metrics:prometheus_api_request unit: prom-counter
# TYPE Gosoline:dev:metrics:prometheus_api_request counter
gosoline:dev:metrics:prometheus_api_request{handler="cur"} 4
gosoline:dev:metrics:prometheus_api_request{handler="decr"} 4
gosoline:dev:metrics:prometheus_api_request{handler="incr"} 5
# HELP Gosoline:dev:metrics:prometheus_important_counter unit: Count
# TYPE Gosoline:dev:metrics:prometheus_important_counter counter
gosoline:dev:metrics:prometheus_important_counter 25
```
