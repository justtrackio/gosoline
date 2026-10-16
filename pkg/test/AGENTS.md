# Test Utilities Package Agent Guide

## Scope
- Provides shared testing helpers: assertions, suites, env management, and matchers used across packages.
- Abstracts Docker-based integration environments and context matchers.

## Key directories
- `assert/` - custom testify extensions and convenience helpers.
- `env/` - spin up local stacks (e.g., Redis, DynamoDB, Floci) for integration tests.
- `matcher/` - gomock/testify matchers (context, time, slices).
- `suite/` - base suites for integration/functional tests with setup/teardown hooks.

## Common tasks
- Add matcher: extend `matcher/` and update README/examples so developers know when to use it.
- Enhance env providers: modify `env/` to support new services; document required Docker images.
- Update base suites: extend `suite/` when tests need new lifecycle hooks (fixtures, tracing, etc.).

## Testing
- `go test ./pkg/test/...` must stay green; it is cheap to run and catches regressions fast.
- When env changes require Docker, run targeted integration suites (e.g., `go test -tags integration,fixtures ./test/...`).

## External services
- Set `test.container_manager.runner_type: external` to connect MySQL, Redis, Mailpit, WireMock and Floci (AWS APIs) to services started by CI. Each component supports `host` and `port`; Mailpit also supports `web_port`. Omitted ports use the service's normal port. Local Docker remains the default runner.
- MySQL gets a separate database per environment. Select separate Redis databases with `test.components.redis.<name>.db` and matching named application client databases when packages share a Redis process. Alternatively, `auto_db: true` assigns each external environment a database via a counter in reserved database 0; named application clients must inherit `redis.default.db`. Configure enough Redis databases for the job; allocation fails when exhausted. Explicit positive database numbers remain caller-managed. WireMock and Mailpit require separate endpoints or sequential suites because their reset/message APIs share server state.
- Floci 2.3.0 (`floci/floci`) is the AWS emulator. Native in-memory implementations use port 4566. Keep the image command: it configures Docker network access. `test.components.floci.default: {}` supplies all AWS clients; `Environment.Floci("default")` exposes DynamoDB, S3, SNS and Toxiproxy helpers. AWS configuration auto-detects Floci unless it is skipped.
- `test.components.dynamodb.default: {}` and `test.components.s3.default: {}` expose client-scoped DynamoDB/S3 components using the same Floci image. Disable Floci auto-detection when using only these components. Both support external `host` and `port`; omitted external ports use 4566.
- Default DynamoDB table patterns get a per-environment prefix, including patterns from environment variables. S3 isolates explicit and implicit default blob buckets while preserving shared buckets and store prefixes. Test-owned names use per-config overrides, never process-global environment changes or server-wide resets. Explicit repository and named-client naming/endpoint overrides remain caller-owned isolation.
- Floci covers API integration, not AWS timing, throttling, IAM or service guarantees.

## Common matchers
```go
import "github.com/justtrackio/gosoline/pkg/test/matcher"

// Match any context
mock.EXPECT().Method(matcher.Context).Return(nil)

// Match specific time
mock.EXPECT().Method(matcher.Time(expectedTime)).Return(nil)
```

## Suite pattern
```go
import "github.com/justtrackio/gosoline/pkg/test/suite"

type MySuite struct {
    suite.Suite
}

func (s *MySuite) SetupSuite() {
    // Start containers, load fixtures
}

func (s *MySuite) TearDownSuite() {
    // Stop containers
}

func TestMySuite(t *testing.T) {
    suite.Run(t, new(MySuite))
}
```

## Environment helpers
Use `suite.WithAppOptions(...)` to configure applications under test with application options, such as
`application.WithLoggerContextFieldsMessageEncoder` for global logger context propagation through stream messages.

| Helper | Package | Purpose |
|--------|---------|--------|
| `env/` | Floci, Redis, MySQL | Docker container management |
| `assert/` | Custom assertions | Extended testify assertions |

## Tips
- Keep helper APIs backward compatible—every package imports `pkg/test`.
- Avoid leaking goroutines from env helpers; always shut down containers in `TearDownSuite`.
- Document new env variables or required binaries in this file so other agents can reproduce setups.
