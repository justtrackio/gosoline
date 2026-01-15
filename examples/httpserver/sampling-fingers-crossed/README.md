# HTTP server example: sampling + fingers-crossed logging

This example demonstrates how to enable gosoline's context-based sampling and fingers-crossed logging in an HTTP service.

## Run

```bash
go run .
```

Server listens on `http://127.0.0.1:8088`.

## Endpoints

- `GET /success`: returns HTTP 200
- `GET /fail`: returns HTTP 500 (useful to see fingers-crossed flushing)

## Try it

### 1) Normal request

```bash
curl -i http://127.0.0.1:8088/success
```

### 2) Force sampling via request header

```bash
curl -i -H 'X-Goso-Sampled: true' http://127.0.0.1:8088/success
curl -i -H 'X-Goso-Sampled: false' http://127.0.0.1:8088/success
```

### 3) Trigger a failure (flushes buffered logs)

```bash
curl -i -H 'X-Goso-Sampled: false' http://127.0.0.1:8088/fail
```

On a non-sampled request, gosoline buffers most log lines and flushes them on errors and failed requests.
