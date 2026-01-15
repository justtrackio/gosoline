---
slug: sampling-fingers-crossed-logging
title: "Log Sampling in Go: Less Noise, More Debuggability"
authors: [jaka]
tags: [logging, sampling, observability]
---

High-traffic services have a logging problem: the more successful traffic you handle, the more you pay to store and query logs that rarely matter. But if you turn logging down, you lose the context you need when something *does* break.

{/* truncate */}

## Why traditional log sampling isn’t enough

Many production setups already try to reduce cost by doing *log pipeline sampling* in tools like Fluentd/Fluent Bit, Logstash, or the logging backend itself.

This approach usually looks like “drop X% of log lines” or “keep 1 out of N entries”, often applied uniformly across an application or per log level.

That can help with volume, but it has two fundamental drawbacks:

- **No context-based sampling**: pipeline sampling typically operates on individual log events. It doesn’t understand that a set of logs belong to the same HTTP request, stream message, or job run. You can easily end up keeping the exception log but dropping the ten lines that explain *why* it happened.
- **No fingers-crossed behavior**: pipeline sampling drops data permanently. It can’t buffer the “boring but useful if it fails” debug logs and only emit them when a failure occurs.

In other words: traditional sampling reduces cost, but it tends to reduce *debuggability* at the same time.

The goal of the feature set in this post is to reduce log volume while **keeping failure context intact**.

A good logging system should give you:

- Low log volume for routine success paths
- Full, high-fidelity context for failures
- A way to “turn up” verbosity for a small, controlled slice of traffic
- Consistent behavior across HTTP requests, background jobs, and message consumers

This post walks through two techniques that work especially well together:

1. **Sampling** — make a consistent sampled/not-sampled decision and store it in `context.Context`.
2. **Fingers-crossed logging** — buffer logs and only flush them when an error occurs.

The examples use gosoline (a Go framework for cloud microservices), but the patterns are transport-agnostic and apply to any Go service architecture.

---

## The two ideas

### 1) Sampling is a decision carried in context

Sampling isn’t “randomly drop log lines”. It’s a *decision about a unit of work*:

- Sampled: treat this request/message/job as “debuggable”, allow verbose logs
- Not sampled: keep things quiet unless there is a failure

The important part: store that decision in `context.Context`, so anything downstream can read it.

In gosoline, this lives in the `smplctx` package.

### 2) Fingers-crossed logging buffers context and only emits on failure

“Fingers-crossed” logging means:

- Buffer `debug/info/warn` messages during a scope
- If the scope ends successfully: discard the buffer
- If an error happens: flush the full buffer (the story leading up to the error)

That gives you rich “what happened before it failed?” context—without paying for it on every success.

---

## The behavior model (simple mental map)

Think in terms of *two* questions:

1) Is this unit of work marked “sampled”?  
2) Did it fail?

If it’s sampled, you log as usual.
If it’s not sampled, you buffer; if it fails, you flush.

---

## How to enable it in a gosoline service

### How to configure sampling

There are two separate things to set up:

- **How decisions are made** (strategies): controlled by config (`sampling.enabled`, `sampling.strategies`, and strategy-specific settings).
- **What gosoline does with that decision** (behavior): enabled via an application option (next section).

This separation is deliberate: it lets you define and tune your sampling strategies without changing code, while still keeping sampling-dependent behavior opt-in at the application wiring level.

Here’s a minimal config that enables sampling and selects strategies:

```yaml
sampling:
  enabled: true
  strategies:
    - tracing
```

---

### How to enable sampling in the application

The config above only describes **how to decide** whether something should be sampled.

To actually **activate sampling-aware behavior** (fingers-crossed logging and sampling propagation across messages), enable it in the application wiring:

```go
app.Run(
  // ...
  app.WithSampling,
)
```

Why is this a separate switch?

- It keeps the behavior opt-in. Many services may want to create sampling decisions (e.g. for metrics or tracing consistency) without changing logging/stream behavior.
- It lets you roll out safely: you can ship configs ahead of time, then enable the new behavior explicitly in code when you’re ready.

What `app.WithSampling` does at runtime:

- **Logger integration**: enables sampling-aware logging so the logger can react to `sampled=true|false` on the context.
- **Stream integration**: propagates the sampling decision as a message attribute so consumers can keep consistent behavior across service boundaries.

---

### Strategy guide (how strategies decide)

#### `tracing`: follow upstream trace sampling (X-Ray / OpenTelemetry)

In modern setups, incoming requests often already carry a sampling decision as part of distributed tracing.

- **AWS X-Ray** propagates trace context via the `X-Amzn-Trace-Id` header. The header can include a sampling decision (for example, `Sampled=1`), indicating that this request should be traced.
- **OpenTelemetry** commonly propagates trace context via the W3C `traceparent` header (and optionally `tracestate`). In W3C propagation, the *trace flags* include a “sampled” bit.

The `tracing` sampling strategy reuses that existing decision:

- If there is valid tracing information on the context, gosoline treats the trace sampling flag as the source of truth.
- This keeps logs and traces aligned: if a request is traced (sampled), you typically also want its logs to be available immediately and in higher detail.

This is especially useful when your sampling decision is made upstream (e.g., an ingress, API gateway, or service mesh) and you want every downstream service to follow the same “sampled vs. not sampled” choice.

#### `probabilistic`: guaranteed baseline + extra traffic

If you don’t have tracing (or don’t want to couple sampling to tracing), a probabilistic strategy is a good default.

It guarantees at least a small amount of sampled traffic per time window (so you always have some detailed examples), and then optionally samples additional requests at a configured percentage.

Configuration example:

```yaml
sampling:
  enabled: true
  strategies:
    - probabilistic
  settings:
    probabilistic:
      interval: 1s
      fixed_sample_count: 1
      extra_rate_percentage: 5
```

Interpretation:

- guarantee at least `fixed_sample_count` sampled decisions per `interval`
- additionally sample `extra_rate_percentage`% of other traffic

This is a nice operational compromise: you always have *some* detailed traffic, and you can raise/lower the extra percentage depending on cost and debugging needs.

### multiple strategies

You can also combine multiple strategies. A common pattern is:

- Use `tracing` when trace context exists.
- Fall back to `probabilistic` when there is no trace (so you still get a small, controlled amount of sampled traffic).

Example:

```yaml
sampling:
  enabled: true
  strategies:
    - tracing
    - probabilistic
  settings:
    probabilistic:
      interval: 1s
      fixed_sample_count: 1
      extra_rate_percentage: 5
```

---

### How the sampling decision is forwarded downstream in stream messages

In real systems, the most painful sampling failures happen at service boundaries:

- Service A decides “this request is sampled” (maybe because tracing says so).
- Service A publishes an event.
- Service B consumes the event… and makes its *own* sampling decision.

If Service B decides differently, you end up with inconsistent observability: a trace might exist for the end-to-end operation, but the logs are only high-fidelity in some of the services.

To avoid that, gosoline can **propagate the sampling decision via stream message metadata**.

When `app.WithSampling` is enabled, gosoline adds a default stream encode handler that:

- Reads the current sampling decision from the context.
- Writes it to a message attribute named `sampled`.

On the consumer side, gosoline restores that decision from the incoming message attribute and places it on the consumer’s `context.Context`. That restored decision should take precedence over locally configured strategies.

The effect is that a single upstream “sampled vs not sampled” choice (often driven by tracing) stays consistent across the entire chain of producers and consumers, without downstream services having to re-decide or accidentally choose differently.

---

## How fingers-crossed works in HTTP servers and stream consumers

Fingers-crossed is easiest to reason about if you think of it as **request/message-scoped buffering**:

- A scope provides a buffer “attached” to the `context.Context`.
- While the scope is active and the context is **not sampled**, non-error log calls are collected in that buffer.
- When a failure signal happens, the buffer is flushed in order, so you see the full lead-up.

### HTTP server behavior

In gosoline’s `httpserver`, the middleware chain is set up so that:

1. A sampling decision is made early in the request lifecycle.
2. The logging middleware establishes a fingers-crossed scope for the request context.
3. At the end of the request, gosoline decides whether to flush based on the **HTTP status code**.

What you get in practice:

- **Sampled request** (`sampled=true`): logs are written immediately (no buffering).
- **Not sampled request** (`sampled=false`): non-error logs are buffered.
- **If an error is logged**: the buffer is flushed immediately, so you get the lead-up right when the first error happens.
- **Otherwise, at the end of the request**:
  - **status < 400**: buffered logs are discarded (you don’t pay for successful traffic).
  - **status >= 400**: buffered logs are flushed, giving you the debug trail for failed requests.

This makes failures “speak up” with context, without turning normal traffic noisy.

### Stream consumer behavior

Stream consumers don’t have an HTTP status code, so they need a different “failure signal”. In gosoline, the model is:

- Consumers restore the sampling decision from the incoming message (attribute `sampled`) when present.
- When the context is **not sampled**, gosoline buffers non-error logs while processing the message.
- The buffer is flushed when an **error-level log** happens, or when you explicitly call `log.FlushFingersCrossedScope(ctx)`.

Practical implications:

- For a successful message, not-sampled logs stay quiet.
- For a failing message, the error log causes the preceding buffered info/debug logs to be emitted as well.

That gives you “quiet success, loud failure” behavior across both HTTP and messaging, while still keeping decisions consistent end-to-end.

---

## How to use “fingers-crossed” logging manually (jobs, CLIs, workers)

For non-HTTP code (CLI, cron job, custom worker), you can wrap work in a fingers-crossed scope:

```go
ctx := context.Background()

// Mark this work as not sampled, to demonstrate buffering behavior.
ctx = smplctx.WithSampling(ctx, smplctx.Sampling{Sampled: false})

// Create a buffer scope. Logs won’t emit immediately.
ctx = log.WithFingersCrossedScope(ctx)

logger.Info(ctx, "starting job: %s", jobID)
logger.Debug(ctx, "details: %+v", payload)

// When an error is logged, the buffer flushes.
if err := doWork(ctx); err != nil {
  logger.Error(ctx, "job failed: %w", err)
  return
}
```

If you want explicit control, you can flush manually:

```go
log.FlushFingersCrossedScope(ctx)
```

---

## How to do HTTP request sampling (including a “force sample” override)

If you want a runnable example, see [`examples/httpserver/sampling-fingers-crossed`](https://github.com/justtrackio/gosoline/tree/main/examples/httpserver/sampling-fingers-crossed).

At runtime, you often want a quick escape hatch: “sample this one request” or “don’t sample this request”.

Gosoline’s HTTP server installs sampling middleware early, and it supports an override header:

- `X-Goso-Sampled: true` or `false` (parsed as a boolean)

So you can do:

```bash
curl -H 'X-Goso-Sampled: true' https://service.example/api/expensive-call
```

In sampled requests, logs behave normally (written immediately).

In not-sampled requests, gosoline uses a fingers-crossed scope: logs are buffered and then flushed automatically when the request fails (HTTP status `>= 400`). This is particularly effective for 500s: you get the full pre-error story without turning the whole service noisy.

---

## Rollout guidance (practical tips)

A few things that make this go smoothly:

- Start with low sampling rates (or tracing-based sampling), then tune.
- Decide what “failure” means for your system. For HTTP it’s natural to flush on `>= 400`, but if your APIs return 4xx for expected control-flow, you may want to review that behavior and/or your status conventions.
- Treat sampling as an *observability strategy*, not just a logging trick: it works best when logs, tracing, and message processing all use the same “sampled/not-sampled” decision.

---

## Closing: why this is worth it

This combo is a good fit for Go services because it leans into Go’s strengths:

- `context.Context` is already the idiomatic carrier of request-scoped state
- middleware boundaries (HTTP, consumers, jobs) are natural points to decide and propagate sampling
- you can keep production cheap and quiet, while still having deep debugging data when it matters
