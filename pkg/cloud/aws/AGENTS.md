# AWS Cloud Package Agent Guide

## Scope
- Houses all AWS integrations (SQS, SNS, Kinesis, DynamoDB, S3, ECS, etc.).
- Wraps AWS SDK v2 clients with gosoline configuration, logging, retry, and naming helpers.
- Provides credentials resolution for local dev, tests, and deployed runtimes.

## Key areas
- Client factories live beside each service (e.g., `sqs/`, `sns/`, `kinesis/`).
- Shared helpers (middleware, retry, credentials) at package root: `awsv2*.go`, `credentials*.go`, `error.go`.
- Naming uses `cfg.Identity.Format()` with pattern-based macros; service tests assert patterns under each subpackage's `*_test.go`.

## Common tasks
- Adding a service: create `pkg/cloud/aws/<service>` with client settings struct, factory, naming helpers, and unit tests following SQS/SNS patterns.
- Adjusting retries/backoff: edit `awsv2_retry.go` and keep unit tests in `awsv2_test.go` updated.
- Credential flows: modify `credentials_default*.go` only after checking impacts on integration tests under `pkg/cloud/aws/*` and `examples/cloud`.

## Testing
- Service-specific: `go test ./pkg/cloud/aws/<service>`.
- Shared helpers: `go test ./pkg/cloud/aws`.
- For changes touching naming/macros, also run `go test ./pkg/stream`.

## Service subpackages
| Package | Purpose | Config prefix |
|---------|---------|---------------|
| `athena/` | Athena query client | `cloud.aws.athena` |
| `cloudwatch/` | Metrics/logs export | `cloud.aws.cloudwatch` |
| `dynamodb/` | Low-level DDB client | `cloud.aws.dynamodb` |
| `ec2/` | Instance metadata | `cloud.aws.ec2` |
| `ecs/` | Container metadata | `cloud.aws.ecs` |
| `glue/` | Glue Data Catalog | `cloud.aws.glue` |
| `kinesis/` | Stream client, naming | `cloud.aws.kinesis` |
| `rds/` | RDS client | `cloud.aws.rds` |
| `resourcegroupstaggingapi/` | Resource tagging API | `cloud.aws.resourcegroupstaggingapi` |
| `s3/` | Object storage | `cloud.aws.s3` |
| `secretsmanager/` | Secrets retrieval | `cloud.aws.secretsmanager` |
| `servicediscovery/` | Cloud Map service discovery | `cloud.aws.servicediscovery` |
| `ses/` | Email sending | `cloud.aws.ses` |
| `sns/` | Topic client, naming | `cloud.aws.sns` |
| `sqs/` | Queue client, naming | `cloud.aws.sqs` |
| `ssm/` | Parameter store | `cloud.aws.ssm` |

## Naming patterns
AWS services (SQS, SNS, Kinesis) use `cfg.Identity.Format()` with pattern-based macros:

| Placeholder | Description |
|-------------|-------------|
| `{app.env}` | Environment |
| `{app.name}` | Application name |
| `{app.tags.<key>}` | Any tag value (dynamic, e.g., project, family, group, region) |
| `{queueId}` | SQS queue identifier |
| `{topicId}` | SNS topic identifier |
| `{streamName}` | Kinesis stream name |

**Note:** DynamoDB table naming uses `ModelId` (from `pkg/ddb`), not `cfg.Identity.Format()` directly.

## Tips
- Keep naming patterns using `cfg.Identity.Format()` macros (`{app.tags.<key>}`, etc.)—never introduce new placeholder names without updating documentation.
- Each service subpackage usually needs fixture-backed tests; mock AWS SDK clients with generated mocks from `.mockery.yml`.
- Avoid hard-coding regions or account IDs; rely on config keys documented in root `AGENTS.md`.
