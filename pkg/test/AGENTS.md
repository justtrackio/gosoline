# Test Utilities Package Agent Guide

## Scope
- Provides shared testing helpers: assertions, suites, env management, and matchers used across packages.
- Abstracts Docker-based integration environments and context matchers.

## Key directories
- `assert/` - custom testify extensions and convenience helpers.
- `env/` - spin up local stacks (e.g., Redis, DynamoDB, Localstack) for integration tests.
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
- Set `test.container_manager.runner_type: external` to connect MySQL, Redis, Mailpit, WireMock, DynamoDB Local and S3 to services started by CI. Each component supports `host` and `port`; Mailpit also supports `web_port`. Omitted ports use the service's normal port. Local Docker remains the default runner.
- MySQL gets a separate database per environment. Select separate Redis databases with `test.components.redis.<name>.db` and matching named application client databases when packages share a Redis process. WireMock and Mailpit require separate endpoints or sequential suites because their reset/message APIs share server state.
- Declare `test.components.dynamodb.default: {}` for standalone DynamoDB Local (`amazon/dynamodb-local:3.1.0`, `-inMemory -sharedDb -disableTelemetry`). It prefixes the default table naming pattern per environment and purges owned tables only. Explicit repository or named-client table naming overrides must provide their own isolation.
- Declare `test.components.s3.default: {}` for S3 API tests backed by `motoserver/moto:5.2.3`. Configured blob buckets receive an environment-specific suffix; store prefixes and shared-bucket relationships are preserved. DynamoDB and S3 endpoints are scoped to their AWS clients, and no server-wide reset is used. Explicit custom client endpoints/bucket overrides must provide their own isolation.
- Disable LocalStack auto-detection when using standalone AWS components. These emulators cover API integration, not AWS timing, throttling, IAM or service guarantees.

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
| `env/` | LocalStack, Redis, MySQL | Docker container management |
| `assert/` | Custom assertions | Extended testify assertions |

## Tips
- Keep helper APIs backward compatible—every package imports `pkg/test`.
- Avoid leaking goroutines from env helpers; always shut down containers in `TearDownSuite`.
- Document new env variables or required binaries in this file so other agents can reproduce setups.
