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

## GitHub MCP server workflow
- **Repository:** `justtrackio/gosoline`. Pass owner `justtrackio` and name `gosoline` to GitHub MCP tools.
- **Search issues/PRs:** Use `github-pull-request_formSearchQuery` to convert natural language to GitHub search syntax, then execute with `github-pull-request_doSearch`. Filter by state, labels, or assignees as needed.
- **Inspect an issue:** Fetch issue details with `github-pull-request_issue_fetch` by supplying `repo: {owner: "justtrackio", name: "gosoline"}` and the `issueNumber`.
- **Active PR context:** Use `github-pull-request_activePullRequest` to get details about the currently checked-out PR, including title, description, changed files, review comments, and CI status.
- **Suggest fixes:** Use `github-pull-request_suggest-fix` to analyze an issue and propose implementation approaches.
- **Pull requests:** Do not create PRs yourself. Use `github-pull-request_openPullRequest` to inspect the currently visible PR if needed for review context.

## Git contribution rules
- **Branch creation:** Never create branches manually. Let maintainers handle branch creation.
- **Commits:** Do not run `git commit` locally. Keep your workspace uncommitted so CI and reviewers can inspect the full diff without extra history.
- **Pull requests:** Do not create PRs yourself. Ask a maintainer to open the PR and check the branch out locally.
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

### AppIdentity macros (cfg package)
Used in queue/topic/stream/namespace names via `cfg.NamingTemplate`:
- `{app.env}` - environment from `app.env` config
- `{app.name}` - application name from `app.name` config
- `{app.tags.<key>}` - any tag value (fully dynamic)

Tags are fully dynamic - common examples include `project`, `family`, `group`, but any tag key is supported (e.g., `{app.tags.region}`, `{app.tags.team}`, `{app.tags.costCenter}`).

### Resource-specific placeholders
Each service adds its own resource identifiers:
- SQS: `{queueId}`
- SNS: `{topicId}`
- Kinesis: `{streamName}`
- Kafka: `{topicId}`, `{groupId}`
- Redis: `{name}` (redis client name)

### ModelId macros (mdl package)
Used in DynamoDB table names via `mdl.ModelId.ReplaceMacros(pattern)`:
- `{project}`, `{env}`, `{family}`, `{group}`, `{app}` - from ModelId fields
- `{modelId}` - the model's name

**Canonical Model IDs (`app.model_id.domain_pattern`):**
For canonical model IDs (used in message routing, etc.), the pattern works differently:
- It supports standard `{app.env}`, `{app.name}`, and `{app.tags.*}` placeholders
- `{modelId}` is **NOT** used; the model name is automatically appended as the last segment
- Example pattern: `{app.tags.project}.{app.env}` -> `myProject.production.myModel`

Note: DynamoDB table naming uses ModelId-based macros (legacy style), not AppIdentity macros.

### Example configs
```yaml
# SQS queue naming
cloud.aws.sqs.clients.default.naming.pattern: "{app.tags.project}-{app.env}-{app.tags.family}-{app.tags.group}-{queueId}"

# SNS topic naming
cloud.aws.sns.clients.default.naming.pattern: "{app.tags.project}-{app.env}-{app.tags.family}-{app.tags.group}-{topicId}"

# Kinesis stream naming
cloud.aws.kinesis.clients.default.naming.pattern: "{app.tags.project}-{app.env}-{app.tags.family}-{app.tags.group}-{streamName}"

# Kafka topic naming
kafka.naming.topic_pattern: "{app.tags.project}-{app.env}-{app.tags.family}-{app.tags.group}-{topicId}"

# CloudWatch namespace
metric.writer.cloudwatch.naming.pattern: "{app.tags.project}/{app.env}/{app.tags.family}/{app.tags.group}-{app.name}"
```

### Strict placeholder validation
Unknown placeholders in naming patterns return an error. Allowed placeholders are:
- Fixed identity: `{app.env}`, `{app.name}`
- Dynamic tags: any `{app.tags.<key>}` where `<key>` is non-empty
- Resource-specific: as registered by each service (e.g., `{queueId}`, `{topicId}`)

This prevents typos like `{app.tag.project}` (missing 's') or old-style `{project}` from silently failing.

### Pattern-driven tag requirements
Tags are only required if the naming pattern uses them. For example:
- Pattern `{app.env}-{queueId}` does NOT require any tags
- Pattern `{app.tags.project}-{app.env}-{queueId}` requires only the `project` tag
- Pattern `{app.tags.region}-{app.tags.team}-{app.env}` requires `region` and `team` tags

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
