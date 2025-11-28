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
| Helper | Package | Purpose |
|--------|---------|--------|
| `env/` | LocalStack, Redis, MySQL | Docker container management |
| `assert/` | Custom assertions | Extended testify assertions |

## Tips
- Keep helper APIs backward compatible—every package imports `pkg/test`.
- Avoid leaking goroutines from env helpers; always shut down containers in `TearDownSuite`.
- Document new env variables or required binaries in this file so other agents can reproduce setups.
