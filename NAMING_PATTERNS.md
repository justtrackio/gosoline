# Naming Patterns in Gosoline

Gosoline uses a unified, configuration-driven approach for naming resources (SQS queues, SNS topics, DynamoDB tables, etc.) and other identifiers. This system relies on `AppIdentity` configuration and a set of macro substitutions.

## Core Concepts

### AppIdentity
Every Gosoline application is identified by its **name**, **environment**, and a set of **tags**. These are defined in your `config.dist.yml` or `config.yml`:

```yaml
app:
  name: my-app
  env: dev
  tags:
    project: my-project
    family: my-family
    group: my-group
    team: my-team
```

### Global Macros
The following macros are available in almost all naming patterns. They resolve directly from the `AppIdentity`:

| Macro | Source Config | Description |
|-------|---------------|-------------|
| `{app.name}` | `app.name` | The application name |
| `{app.env}` | `app.env` | The environment (e.g., dev, prod) |
| `{app.tags.<tag>}` | `app.tags.<tag>` | Any tag defined in `app.tags` |

*Examples:*
- `{app.tags.project}` resolves to `my-project`
- `{app.tags.cost_center}` resolves to the value of `app.tags.cost_center`

## Service-Specific Patterns

Different services support additional macros specific to their context. You can customize the naming pattern for each service in your configuration.

### AWS SQS (Queues)
**Config Key:** `cloud.aws.sqs.clients.<client_name>.naming.pattern`
**Default:** `{app.tags.project}-{app.env}-{app.tags.family}-{app.tags.group}-{queueId}`

| Macro | Description |
|-------|-------------|
| `{queueId}` | The logical name of the queue provided in code |

### AWS SNS (Topics)
**Config Key:** `cloud.aws.sns.clients.<client_name>.naming.pattern`
**Default:** `{app.tags.project}-{app.env}-{app.tags.family}-{app.tags.group}-{topicId}`

| Macro | Description |
|-------|-------------|
| `{topicId}` | The logical name of the topic provided in code |

### AWS Kinesis
Kinesis configuration supports naming for both Streams and the DynamoDB Metadata table used by the Kinsumer.

**Stream Config Key:** `cloud.aws.kinesis.clients.<client_name>.naming.stream_pattern`
**Stream Default:** `{app.tags.project}-{app.env}-{app.tags.family}-{app.tags.group}-{streamName}`

| Macro | Description |
|-------|-------------|
| `{streamName}` | The logical name of the stream provided in code |

**Metadata Config Key:** `cloud.aws.kinesis.clients.<client_name>.naming.metadata_pattern`
**Metadata Default:** `{app.env}-kinsumer-metadata`

### AWS CloudWatch (Metrics Namespace)
CloudWatch naming configures the **Namespace** under which metrics are published.

**Config Key:** `metric.writer_settings.cloudwatch.naming.pattern`
**Default:** `{app.tags.project}/{app.env}/{app.tags.family}/{app.tags.group}-{app.name}`

| Macro | Description |
|-------|-------------|
| `{app.env}` | Environment |
| `{app.name}` | Application name |
| `{app.tags.<tag>}` | Any tag from the identity |

### AWS DynamoDB (Tables)
DynamoDB table naming uses the standard `AppIdentity` macros plus `{name}` for the model name.

**Config Key:** `cloud.aws.dynamodb.clients.<client_name>.naming.pattern`
**Default:** `{app.tags.project}-{app.env}-{app.tags.family}-{app.tags.group}-{name}`

| Macro | Description |
|-------|-------------|
| `{app.env}` | Environment |
| `{app.name}` | Application name |
| `{app.tags.<tag>}` | Any tag from the model's identity |
| `{name}` | The model name |

**Note:** Unlike the Canonical Model ID, the table name pattern **does not** automatically append the name. You must include `{name}` in the pattern.

### AWS S3 Buckets
**Config Key:** `cloud.aws.s3.clients.<client_name>.naming.bucket_pattern`
**Default:** `{app.tags.project}-{app.env}-{app.tags.family}`

| Macro | Description |
|-------|-------------|
| `{app.env}` | Environment |
| `{app.name}` | Application name |
| `{app.tags.<tag>}` | Any tag from the identity |
| `{bucketId}` | The bucket ID (e.g. the blob store name) |

**Note:** For the `blob` package, you can still override the bucket name explicitly using `blob.<store_name>.bucket`.

### ModelId Domain Pattern
Resources that use `ModelId` but are not tied to a specific service client (like canonical message routing keys) use the domain pattern.

**Config Key:** `app.model_id.domain_pattern`
**Recommended:** `{app.tags.project}.{app.env}.{app.tags.family}.{app.tags.group}`

This pattern works similarly to DynamoDB naming:
1.  It resolves the pattern using the model's identity tags.
2.  It automatically appends the model name as the final segment (separated by a dot).

**Strict Delimiter Constraint:**
Unlike other naming patterns, the `ModelId` domain pattern **must** use dots (`.`) as delimiters between placeholders. Dashes, underscores, or static text are **not permitted**. This strict format allows the system to parse a string representation back into a `ModelId` object.

*   ✅ `{app.tags.project}.{app.env}`
*   ❌ `{app.tags.project}-{app.env}` (Invalid delimiter)

**Example:**
*   Pattern: `{app.tags.project}.{app.env}`
*   Model Identity: `env=prod`, `project=logistics`, `name=Shipment`
*   Result: `logistics.prod.Shipment`

### Kafka
Kafka supports naming for both Topics and Consumer Groups.

**Topic Config Key:** `kafka.naming.topic_pattern`
**Topic Default:** `{app.tags.project}-{app.env}-{app.tags.family}-{app.tags.group}-{topicId}`

| Macro | Description |
|-------|-------------|
| `{topicId}` | The logical topic name |

**Consumer Group Config Key:** `kafka.naming.group_pattern`
**Group Default:** `{app.tags.project}-{app.env}-{app.tags.family}-{app.tags.group}-{app.name}-{groupId}`

| Macro | Description |
|-------|-------------|
| `{groupId}` | The logical consumer group ID |

### Redis
Redis has patterns for both the server address (for service discovery) and key namespacing.

**Address Config Key:** `redis.<client_name>.naming.address_pattern`
**Address Default:** `{name}.{app.tags.group}.redis.{app.env}.{app.tags.family}`

| Macro | Description |
|-------|-------------|
| `{name}` | The client name (e.g., "default", "cache") |

**Key Config Key:** `redis.<client_name>.naming.key_pattern`
**Key Default:** `{key}` (often configured to include namespaces like `{app.name}-{key}`)

| Macro | Description |
|-------|-------------|
| `{key}` | The specific key being accessed |

## Configuration Example

Here is how you might configure these patterns in your `config.yml` to enforce a company-wide naming convention:

```yaml
app:
  project: ordering
  env: production
  tags:
    region: eu-central-1
    team: logistics

cloud:
  aws:
    sqs:
      clients:
        default:
          naming:
            # Result: eu-central-1-logistics-ordering-production-myqueue
            pattern: "{app.tags.region}-{app.tags.team}-{app.tags.project}-{app.env}-{queueId}"
```
