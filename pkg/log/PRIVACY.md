# Confidential diagnostics

Keep method, route, status, timing and a safe operation/error code. Use explicit
credential ownership rather than treating every configuration string as secret.

## Values and propagated errors

`log.Secret(value)` masks a value during fmt formatting (including `%+v` and
`%#v`) and JSON encoding. Convert it to a plain string only where the actual
credential is needed. It is not an encrypted storage type.

Wrap provider errors at the boundary that knows their confidential inputs:

```go
err := log.RedactError(providerErr, apiKey)
return fmt.Errorf("authenticating with provider: %w", err)
```

The application error retains its original cause for `errors.Is`/`errors.As`.
Its message masks explicit credential values and supported URL/base64/quoted
forms at token boundaries. Diagnostic identifiers such as `refresh_token` remain
readable when a secret only matches part of that identifier. Pass credentials,
not ordinary configuration values, to this API.

The logger uses `SafeError` to produce a detached export copy before sampling
buffers and sinks. Sentry therefore cannot unwrap that copy to recover the
original confidential error. Keep the application error for control flow; the
export copy deliberately has no original cause. Error chains beyond the privacy
depth limit are omitted rather than exported unsafely.

Known credential keys in common structured maps/slices are masked by
`SanitizeFields`, which copies rather than modifies caller-owned containers.
Raw request/configuration structs and arbitrary unlabeled strings are outside
this inference. Prefer a small diagnostic projection. The `nosecrets` analyzer
in go-linters rejects obvious credential-bearing diagnostic construction; it is
not whole-program taint analysis and does not replace these runtime boundaries.

## HTTP client

The client sanitizes diagnostic URLs and protects errors using declared auth,
API-key and sensitive-query values. `WithBasicAuth`, `WithAuthToken` and sensitive
headers register their credentials. For custom payloads, use
`request.WithSensitiveValues(values...)`. Values sent over the wire are unchanged.
`SanitizeURL` removes userinfo, all query values and fragments; credentials in
arbitrary path segments still require an explicit confidential value.

## HTTP request logs

Both the embedded HTTP server and the extracted `gosoline-project/httpserver`
use route templates; unmatched paths are represented as `[unmatched]`. Raw
queries, referers and raw bodies are omitted by default. Bind-error logs contain
field/rule information or the parser error type, never rejected field values.

Safe diagnostic capture can be enabled explicitly:

```yaml
httpserver:
  default:
    logging:
      request_query_parameters: [page, limit]
      request_headers: [X-Frontend-Version]
      request_referer: false
      request_body: true
      request_body_fields: [operation]
```

Credential-named keys stay masked even if allowlisted. Body capture requires
both `request_body` and `request_body_fields`; non-JSON or malformed bodies have
no raw fallback. `request_body_base64` encodes only the sanitized projection.
An opted-in referer is sanitized. Request parsing and handler input are unchanged.
Choose allowlisted fields whose values and paths cannot carry credentials.

This changes existing diagnostic output: raw path/query fields disappear and
`request_body: true` alone no longer captures the payload. Update dashboards and
log assertions to use route templates and explicit safe projections.

## Adoption and verification

Release Gosoline first, then update the extracted server's Gosoline pin and
release it. Consumers must update the server they actually use; an embedded
server change alone does not protect an extracted-server consumer. Release the
linter bundle independently and update consumer pins. Until release, a local Go
workspace can test the coordinated library changes without inventing a version.

Use synthetic secrets to inspect serialized JSON/text/Sentry output and wrapped
errors, including failed requests and buffered-log flushes. Keep deployed proxy,
trace and telemetry policies aligned. This work does not change authentication
authority or token issuance; Zitadel migration keeps its separate credential
ownership fence.
