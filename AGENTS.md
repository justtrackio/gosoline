# Agent Handbook

Operate in this repository as the maintainer of the **gosoline** application framework—a Go-based toolkit for building cloud microservices with first-class AWS integration.

## Quick facts
- **Module path:** `github.com/justtrackio/gosoline`
- **Go toolchain:** 1.24 (see `.tool-versions` for exact patch level)
- **Primary frameworks:** Gin (HTTP), AWS SDK v2, testify, mockery
- **Key tags:** most packages ship `fixtures` and `integration` build tags. Always include both when building, testing, or linting to pull in fixture wiring.

## Repository layout
- `pkg/`: Core framework packages (50+ packages covering application lifecycle, configuration, logging, AWS integrations, streaming, HTTP server, database, caching, etc.)
- `docs/`: Docusaurus documentation site. Run `cd docs && yarn build` to verify.
- `examples/`: Sample applications demonstrating gosoline features and best practices.
- `test/`: Integration and E2E test suites (blob, cloud, db, db-repo, ddb, fixtures, guard, httpserver, mdlsub, stream, suite). Requires Docker.
- `.github/workflows/`: CI pipelines—mockery check, build, golangci-lint, unit tests, race tests, integration tests.

## Day-to-day workflow for changes
1. Capture user requirements and convert them into a todo list (use the `manage_todo_list` tool). Keep a single active item at a time.
2. Survey the relevant package (readme, docs, nested AGENTS.md) before editing code.
3. Edit Go sources and immediately format using `gofumpt -w <files>`.
4. Regenerate artifacts after interface changes:
   - Mocks: `go generate -run='mockery' ./...`
5. Validate locally:
   - `gofumpt -w .`
   - `go build -tags fixtures,integration ./...`
   - `go test -tags fixtures,integration ./...`
   - `golangci-lint run --build-tags integration,fixtures ./...`
6. Check for missing godoc parts for exported types/functions you added or changed.
7. Update AGENTS.md files if your changes affect package structure, APIs, or workflows documented there. Check both the root `AGENTS.md` and any package-specific `AGENTS.md` (e.g., `pkg/<package>/AGENTS.md`).
8. Summarize work with requirement coverage, commands executed, and pending follow-ups. Never stage or commit; CI and reviewers expect clean diffs only.

## GitHub CLI workflow (gh)
- **Repository:** `justtrackio/gosoline`.
- **Auth & status:** `gh auth status` to confirm login.
- **Search issues/PRs:** `gh search issues "<query>" --repo justtrackio/gosoline` and `gh search prs "<query>" --repo justtrackio/gosoline`.
- **Inspect an issue:** `gh issue view <number> --repo justtrackio/gosoline`.
- **Inspect a PR:** `gh pr view <number> --repo justtrackio/gosoline --web` or `--json` for structured output.
- **CI status:** `gh pr checks <number> --repo justtrackio/gosoline`.
- **Pull requests:** Only create PRs when asked to; prefer `gh pr create` with a summary and checklist.

## Git contribution rules
- **Branch creation:** Only create branches when asked to. Else let maintainers handle branch creation.
- **Commits:** Only commit on feature branches, never main or master.
- **Pull requests:** Only create PRs when asked to.
- **Merging:** Never merge PRs yourself. Leave merges to maintainers or automated pipelines.

## Command reference
| Task | Command |
|------|---------|
| Format | `gofumpt -w .` |
| Build | `go build -tags fixtures,integration ./...` |
| Unit tests | `go test ./...` |
| Unit tests (race) | `go test -race ./...` |
| Integration tests | `go test -tags integration,fixtures ./test/...` |
| Lint | `golangci-lint run --build-tags integration,fixtures ./...` |
| Mock generation | `go generate -run='mockery' ./...` |
| Targeted test | `go test -tags fixtures,integration ./test/<package>/... -run <TestName>` |
| Docs build | `cd docs && yarn install && yarn build` |

## Domain map

### Core packages (`pkg/`)
| Package | Purpose |
|---------|---------|
| `application/` | Application entry point, lifecycle management |
| `kernel/` | Module orchestration, startup/shutdown sequencing |
| `cfg/` | Configuration management, AppId, macro interpolation |
| `log/` | Structured logging infrastructure |
| `httpserver/` | Gin-based HTTP server, middleware, handlers |
| `stream/` | Message streaming, consumers, producers |
| `mdl/` | Model definitions, ModelId |
| `mdlsub/` | Model subscription patterns |

### AWS integrations (`pkg/cloud/aws/`)
| Package | AWS Service |
|---------|-------------|
| `sqs/` | Simple Queue Service |
| `sns/` | Simple Notification Service |
| `kinesis/` | Kinesis Data Streams |
| `dynamodb/` | DynamoDB |
| `s3/` | S3 object storage |
| `cloudwatch/` | CloudWatch metrics/logs |
| `secretsmanager/` | Secrets Manager |
| `ses/` | Simple Email Service |

### Data & persistence
| Package | Purpose |
|---------|---------|
| `db/` | Database connections, migrations |
| `db-repo/` | Repository pattern for SQL |
| `ddb/` | DynamoDB repositories, naming |
| `redis/` | Redis client, caching |
| `kvstore/` | Key-value store abstraction |
| `blob/` | Blob storage abstraction |
| `fixtures/` | Test fixture loading |

### Utilities
| Package | Purpose |
|---------|---------|
| `exec/` | Retry, backoff, execution helpers |
| `clock/` | Time abstraction for testing |
| `uuid/` | UUID generation |
| `funk/` | Functional utilities (map, filter, etc.) |
| `mapx/` | Map utilities |
| `cast/` | Type casting helpers |
| `encoding/` | Encoding utilities |
| `validation/` | Input validation |

## Naming conventions and resource macros

Gosoline uses a macro system for consistent resource naming across AWS services and data stores.

### AppId macros (cfg package)
Used in queue/topic/stream names via `cfg.AppId.ReplaceMacros(pattern)`:
- `{project}` - app_project config value
- `{env}` - environment
- `{family}` - app_family
- `{group}` - app_group
- `{app}` - app_name

### ModelId macros (mdl package)
Used in DynamoDB table names via `mdl.ModelId.ReplaceMacros(pattern)`:
- All AppId macros above, plus:
- `{modelId}` - the model's string representation (`project.family.group.name`)

### Service-specific macros
- SQS/SNS: `{queueId}`, `{topicId}`
- Kinesis: `{streamName}`
- DynamoDB: `{modelId}`

### Example config
```yaml
cloud.aws.sqs.clients.default.naming.pattern: "{project}-{env}-{family}-{group}-{app}-{queueId}"
```

## Conventions & testing patterns
- File naming: `snake_case.go`
- Exported names: `CamelCase`
- JSON struct tags: `camelCase`
- Config struct tags: `cfg:"key_name"`
- Wrap errors: `fmt.Errorf("<context>: %w", err)`
- Always propagate `context.Context`
- Prefer dependency injection via gosoline configuration modules
- Tests use `github.com/stretchr/testify`, Match `context.Context` arguments with `matcher.Context` from `pkg/test/matcher`
- Keep build tags aligned across source and tests (`//go:build fixtures` / `integration`)
